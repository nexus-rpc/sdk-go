package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// NewCompletionHTTPRequest creates an HTTP request deliver an operation completion to a given URL.
func NewCompletionHTTPRequest(ctx context.Context, url string, completion OperationCompletion) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}
	if err := completion.applyToHTTPRequest(httpReq); err != nil {
		return nil, err
	}

	httpReq.Header.Set(headerUserAgent, userAgent)
	return httpReq, nil
}

// OperationCompletion is input for [NewCompletionHTTPRequest].
// It has two implementations: [OperationCompletionSuccessful] and [OperationCompletionUnsuccessful].
type OperationCompletion interface {
	applyToHTTPRequest(*http.Request) error
}

// OperationCompletionSuccessful is input for [NewCompletionHTTPRequest], used to deliver successful operation results.
type OperationCompletionSuccessful struct {
	// Header to send in the completion request.
	Header http.Header
	// Body to send in the completion HTTP request.
	// If it implements `io.Closer` it will automatically be closed by the client.
	Body io.Reader
}

// NewOperationCompletionSuccessful constructs an [OperationCompletionSuccessful] from a JSONable value.
// Marshals the provided value to JSON using [json.Marshal] and sets the proper Content-Type header.
func NewOperationCompletionSuccessful(v any) (*OperationCompletionSuccessful, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	header := make(http.Header, 1)
	header.Set("Content-Type", contentTypeJSON)

	return &OperationCompletionSuccessful{
		Header: header,
		Body:   bytes.NewReader(b),
	}, nil
}

func (c *OperationCompletionSuccessful) applyToHTTPRequest(request *http.Request) error {
	if c.Header != nil {
		request.Header = c.Header.Clone()
	}
	request.Header.Set(headerOperationState, string(OperationStateSucceeded))
	if closer, ok := c.Body.(io.ReadCloser); ok {
		request.Body = closer
	} else {
		request.Body = io.NopCloser(c.Body)
	}
	return nil
}

// OperationCompletionUnsuccessful is input for [NewCompletionHTTPRequest], used to deliver unsuccessful operation
// results.
type OperationCompletionUnsuccessful struct {
	// Header to send in the completion request.
	Header http.Header
	// State of the operation, should be failed or canceled.
	State OperationState
	// Failure object to send with the completion.
	Failure *Failure
}

func (c *OperationCompletionUnsuccessful) applyToHTTPRequest(request *http.Request) error {
	if c.Header != nil {
		request.Header = c.Header.Clone()
	}
	request.Header.Set(headerOperationState, string(c.State))
	request.Header.Set("Content-Type", contentTypeJSON)

	b, err := json.Marshal(c.Failure)
	if err != nil {
		return err
	}

	request.Body = io.NopCloser(bytes.NewReader(b))
	return nil
}

// CompletionRequest is input for CompletionHandler.CompleteOperation.
type CompletionRequest struct {
	// The original HTTP request.
	HTTPRequest *http.Request
	// State of the operation.
	State OperationState
	// Parsed from request and set if State is failed or canceled.
	Failure *Failure
}

// A CompletionHandler can receive operation completion requests as delivered via the callback URL provided in
// start-operation requests.
type CompletionHandler interface {
	CompleteOperation(context.Context, *CompletionRequest) error
}

// CompletionHandlerOptions are options for [NewCompletionHTTPHandler].
type CompletionHandlerOptions struct {
	// Handler for completion requests.
	Handler CompletionHandler
	// A stuctured logging handler.
	// Defaults to slog.Default().
	Logger *slog.Logger
	// Optional marshaler for marshaling objects to JSON.
	// Defaults to json.Marshal.
	Marshaler func(any) ([]byte, error)
}

type completionHTTPHandler struct {
	baseHTTPHandler
	handler CompletionHandler
}

func (h *completionHTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	completion := CompletionRequest{
		State:       OperationState(request.Header.Get(headerOperationState)),
		HTTPRequest: request,
	}
	switch completion.State {
	case OperationStateFailed, OperationStateCanceled:
		if !isMediaTypeJSON(request.Header.Get("Content-Type")) {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request content type: %q", request.Header.Get("Content-Type")))
			return
		}
		var failure Failure
		b, err := io.ReadAll(request.Body)
		if err != nil {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to read Failure from request body"))
			return
		}
		if err := json.Unmarshal(b, &failure); err != nil {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to read Failure from request body"))
			return
		}
		completion.Failure = &failure
	case OperationStateSucceeded:
		// Nothing to do here.
	default:
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request operation state: %q", completion.State))
		return
	}
	if err := h.handler.CompleteOperation(ctx, &completion); err != nil {
		h.writeFailure(writer, err)
	}
}

// NewCompletionHTTPHandler constructs an [http.Handler] from given options for handling operation completion requests.
func NewCompletionHTTPHandler(options CompletionHandlerOptions) http.Handler {
	if options.Marshaler == nil {
		options.Marshaler = json.Marshal
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	return &completionHTTPHandler{
		baseHTTPHandler: baseHTTPHandler{
			logger: options.Logger,
		},
		handler: options.Handler,
	}
}
