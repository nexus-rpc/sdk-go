package nexus

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/nexus-rpc/sdk-go/nexusapi"
)

// An OperationHandle is used to cancel operations and get their result and status.
type OperationHandle struct {
	// Name of the Operation this handle represents.
	Operation string
	// Handler generated ID for this handle's operation.
	ID     string
	client *Client
}

// Options for [nexusclient.OperationHandle.GetInfo].
type GetInfoOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
}

// GetInfo gets operation information issuing a network request to the service handler.
func (h *OperationHandle) GetInfo(ctx context.Context, options GetInfoOptions) (*nexusapi.OperationInfo, error) {
	if h.ID == "" {
		return nil, errHandleForSyncOperation
	}

	url, err := h.client.joinURL(h.Operation, h.ID)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if options.Header != nil {
		httpReq.Header = options.Header.Clone()
	}

	httpReq.Header.Set(headerUserAgent, UserAgent)
	response, err := h.client.Options.HTTPCaller(httpReq)
	if err != nil {
		return nil, err
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response, body)
	}

	return operationInfoFromResponse(response, body)
}

// GetResultOptions are Options for [nexusclient.OperationHandle.GetResult].
type GetResultOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
	// Boolean indicating whether to wait for operation completion or return the current status immediately.
	Wait bool
}

// GetResult gets the result of an operation, issuing a network request to the service handler.
//
// By default, GetResult returns a nil response immediately and no error after issuing a call if the operation has not
// yet completed.
//
// Callers may set [nexusclient.GetResultOptions.Wait] to true to alter this behavior, causing the client to long poll
// for the result until the provided context deadline is exceeded. When the deadline exceeds, GetResult will return a
// nil response and [context.DeadlineExceeded] error. The client may issue multiple requests until the deadline exceeds
// with a max request timeout of [nexusclient.Options.GetResultMaxRequestTimeout].
//
// ⚠️ If a response is returned, its body must be read in its entirety and closed to free up the underlying connection.
func (h *OperationHandle) GetResult(ctx context.Context, options GetResultOptions) (*http.Response, error) {
	url, err := h.client.joinURL(h.Operation, h.ID, "result")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if options.Header != nil {
		httpReq.Header = options.Header.Clone()
	}
	httpReq.Header.Set(headerUserAgent, UserAgent)

	for {
		response, err := h.client.sendGetOperationResultRequest(ctx, httpReq, options.Wait)
		if err != nil {
			if errors.Is(err, errOperationStillRunning) {
				if options.Wait {
					continue
				} else {
					return nil, nil
				}
			}
			return nil, err
		}
		return response, nil
	}
}

// Options for [nexusclient.OperationHandle.Cancel].
type CancelOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
}

// Cancel requests to cancel an asynchronous operation.
//
// Cancelation is asynchronous and may be not be respected by the operation's implementation.
func (h *OperationHandle) Cancel(ctx context.Context, options CancelOptions) error {
	if h.ID == "" {
		return errHandleForSyncOperation
	}

	url, err := h.client.joinURL(h.Operation, h.ID, "cancel")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url.String(), nil)
	if err != nil {
		return err
	}
	if options.Header != nil {
		httpReq.Header = options.Header.Clone()
	}

	httpReq.Header.Set(headerUserAgent, UserAgent)
	response, err := h.client.Options.HTTPCaller(httpReq)
	if err != nil {
		return err
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusAccepted {
		return newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response, body)
	}
	return nil
}
