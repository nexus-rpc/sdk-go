package nexusclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexusapi"
)

type Options struct {
	ServiceBaseURL string
	HTTPClient     *http.Client
}

type Client struct {
	ServiceBaseURL *url.URL
	HTTPClient     *http.Client
}

func NewClient(options Options) (*Client, error) {
	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	serviceBaseUrl, err := url.Parse(options.ServiceBaseURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		ServiceBaseURL: serviceBaseUrl,
		HTTPClient:     client,
	}, nil
}

type OperationHandle struct {
	id string

	client *Client
	state  nexusapi.OperationState
	// mutually exclusive with failure
	response *http.Response
	failure  nexusapi.Failure
}

func (h *OperationHandle) ID() string {
	return h.id
}

func (h *OperationHandle) State() nexusapi.OperationState {
	return h.state
}

type GetResultOptions struct {
	WaitTimeout time.Duration
}

func (h *OperationHandle) Close() error {
	// Body will have already been closed
	if h.state != nexusapi.OperationStateSucceeded {
		return nil
	}
	return h.response.Body.Close()
}

func (h *OperationHandle) Result(ctx context.Context) (*http.Response, error) {
	switch h.state {
	case nexusapi.OperationStateCanceled, nexusapi.OperationStateFailed:
		return nil, &nexusapi.UnsuccessfulOperationError{State: h.state, Failure: h.failure}
	case nexusapi.OperationStateSucceeded:
		return h.response, nil
	default:
		panic("not implemented")
	}
}

type StartOperationRequest struct {
	Operation   string
	CallbackURL string
	RequestID   string
	Header      http.Header
	Body        io.Reader
}

type UnexpectedResponseError struct {
	Message  string
	Response *http.Response
}

func (e UnexpectedResponseError) Error() string {
	return fmt.Sprintf("%s: %s", e.Response.Status, e.Message)
}

func (c *Client) StartOperation(ctx context.Context, request StartOperationRequest) (*OperationHandle, error) {
	url := c.ServiceBaseURL.JoinPath(request.Operation)
	if request.CallbackURL != "" {
		q := url.Query()
		q.Set(nexusapi.QueryCallbackURL, request.CallbackURL)
		url.RawQuery = q.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url.String(), request.Body)
	if err != nil {
		return nil, err
	}

	if request.Header != nil {
		httpReq.Header = request.Header.Clone()
	}
	if request.RequestID == "" {
		requestIDFromHeader := request.Header.Get(nexusapi.HeaderRequestID)
		if requestIDFromHeader != "" {
			request.RequestID = requestIDFromHeader
		} else {
			request.RequestID = uuid.NewString()
		}
	}
	httpReq.Header.Set(nexusapi.HeaderRequestID, request.RequestID)

	response, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	// Do not close response body here to allow successful result to read it.
	switch response.StatusCode {
	case 200, 204:
		handle := &OperationHandle{
			state:    nexusapi.OperationStateSucceeded,
			response: response,
		}
		return handle, nil
	}

	// Do this once here and make sure it doesn't leak.
	defer response.Body.Close()

	switch response.StatusCode {
	case 201:
		if response.Header.Get(nexusapi.HeaderContentType) != nexusapi.ContentTypeJSON {
			// TODO: caller will not be able to inspect response body
			return nil, UnexpectedResponseError{"invalid response content type", response}
		}
		var info nexusapi.OperationInfo
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(body, &info)
		if err != nil {
			return nil, err
		}
		if info.State != nexusapi.OperationStateRunning {
			return nil, UnexpectedResponseError{fmt.Sprintf("invalid operation state in response info: %v", info.State), response}
		}
		handle := &OperationHandle{
			id:       info.ID,
			state:    info.State,
			response: response,
		}
		return handle, nil
	case 482:
		if response.Header.Get(nexusapi.HeaderContentType) != nexusapi.ContentTypeJSON {
			// TODO: caller will not be able to inspect response body
			return nil, UnexpectedResponseError{"invalid response content type", response}
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		var failure nexusapi.Failure
		if err := json.Unmarshal(body, &failure); err != nil {
			return nil, err
		}

		state := nexusapi.OperationState(response.Header.Get(nexusapi.HeaderOperationState))
		handle := &OperationHandle{
			// TODO: ID
			state:   nexusapi.OperationState(state),
			failure: failure,
		}
		switch state {
		case nexusapi.OperationStateCanceled:
			return handle, nil
		case nexusapi.OperationStateFailed:
			return handle, nil
		default:
			return nil, UnexpectedResponseError{fmt.Sprintf("invalid operation state header: %v", state), response}
		}
	default:
		// TODO: caller will not be able to inspect response body
		return nil, UnexpectedResponseError{"unexpected response code", response}
	}
}
