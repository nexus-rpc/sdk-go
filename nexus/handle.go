package nexus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const getResultContextPadding = time.Second * 5

// An OperationHandle is used to cancel operations and get their result and status.
type OperationHandle struct {
	// Name of the Operation this handle represents.
	Operation string
	// Handler generated ID for this handle's operation.
	ID     string
	client *Client
}

// GetOperationInfoOptions are options for [OperationHandle.GetInfo].
type GetOperationInfoOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
}

// GetInfo gets operation information, issuing a network request to the service handler.
func (h *OperationHandle) GetInfo(ctx context.Context, options GetOperationInfoOptions) (*OperationInfo, error) {
	url := h.client.serviceBaseURL.JoinPath(url.PathEscape(h.Operation), url.PathEscape(h.ID))
	request, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if options.Header != nil {
		request.Header = options.Header.Clone()
	}

	request.Header.Set(headerUserAgent, userAgent)
	response, err := h.client.options.HTTPCaller(request)
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

// GetOperationResultOptions are Options for [OperationHandle.GetResult].
type GetOperationResultOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
	// Duration to wait for operation completion. Zero or negative value implies no wait.
	Wait time.Duration
}

// GetResult gets the result of an operation, issuing a network request to the service handler.
//
// By default, GetResult returns (nil, [ErrOperationStillRunning]) immediately after issuing a call if the operation has
// not yet completed.
//
// Callers may set GetOperationResultOptions.Wait to a value greater than 0 to alter this behavior, causing the client
// to long poll for the result issuing one or more requests until the provided wait period exceeds, in which case (nil,
// [ErrOperationStillRunning]) is returned.
//
// The wait time is capped to the deadline of the provided context. Make sure to handle both context deadline errors and
// [ErrOperationStillRunning].
//
// Note that the wait period is enforced by the server and may not be respected if the server is misbehaving. Set the
// context deadline to the max allowed wait period to ensure this call returns in a timely fashion.
//
// ⚠️ If a response is returned, its body must be read in its entirety and closed to free up the underlying connection.
func (h *OperationHandle) GetResult(ctx context.Context, options GetOperationResultOptions) (*http.Response, error) {
	url := h.client.serviceBaseURL.JoinPath(url.PathEscape(h.Operation), url.PathEscape(h.ID), "result")
	request, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if options.Header != nil {
		request.Header = options.Header.Clone()
	}
	request.Header.Set(headerUserAgent, userAgent)

	startTime := time.Now()
	wait := options.Wait
	for {
		if wait > 0 {
			if deadline, set := ctx.Deadline(); set {
				// Ensure we don't wait longer than the deadline but give some buffer prevent racing between wait and
				// context deadline.
				wait = min(wait, time.Until(deadline)+getResultContextPadding)
			}

			q := request.URL.Query()
			q.Set(queryWait, fmt.Sprintf("%dms", wait.Milliseconds()))
			request.URL.RawQuery = q.Encode()
		} else {
			// We're may reuse the request objects mutliple time and will need to reset the query when wait becomes 0 or
			// negative.
			request.URL.RawQuery = ""
		}

		response, err := h.client.sendGetOperationRequest(ctx, request)
		if err != nil {
			if wait > 0 && errors.Is(err, errOperationWaitTimeout) {
				wait = options.Wait - time.Since(startTime)
				continue
			}
		}
		return response, err
	}
}

// CancelOperationOptions are options for [OperationHandle.Cancel].
type CancelOperationOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
}

// Cancel requests to cancel an asynchronous operation.
//
// Cancelation is asynchronous and may be not be respected by the operation's implementation.
func (h *OperationHandle) Cancel(ctx context.Context, options CancelOperationOptions) error {
	url := h.client.serviceBaseURL.JoinPath(url.PathEscape(h.Operation), url.PathEscape(h.ID), "cancel")
	request, err := http.NewRequestWithContext(ctx, "POST", url.String(), nil)
	if err != nil {
		return err
	}
	if options.Header != nil {
		request.Header = options.Header.Clone()
	}

	request.Header.Set(headerUserAgent, userAgent)
	response, err := h.client.options.HTTPCaller(request)
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
