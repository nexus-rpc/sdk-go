package nexus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

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
	url := h.client.serviceBaseURL.JoinPath(h.Operation, h.ID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if options.Header != nil {
		httpReq.Header = options.Header.Clone()
	}

	httpReq.Header.Set(headerUserAgent, userAgent)
	response, err := h.client.options.HTTPCaller(httpReq)
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
// The wait time is capped to the deadline of the provided context.
//
// ⚠️ If a response is returned, its body must be read in its entirety and closed to free up the underlying connection.
func (h *OperationHandle) GetResult(ctx context.Context, options GetOperationResultOptions) (*http.Response, error) {
	url := h.client.serviceBaseURL.JoinPath(h.Operation, h.ID, "result")
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if options.Header != nil {
		httpReq.Header = options.Header.Clone()
	}
	httpReq.Header.Set(headerUserAgent, userAgent)

	startTime := time.Now()
	for {
		var wait time.Duration
		if options.Wait > 0 {
			wait = options.Wait - time.Since(startTime)
			if wait < 0 {
				return nil, ErrOperationStillRunning
			}
		}
		response, err := h.client.sendGetOperationResultRequest(ctx, httpReq, wait)
		if err != nil {
			if errors.Is(err, ErrOperationStillRunning) && options.Wait > 0 {
				continue
			}
			return nil, err
		}
		return response, nil
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
	url := h.client.serviceBaseURL.JoinPath(h.Operation, h.ID, "cancel")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url.String(), nil)
	if err != nil {
		return err
	}
	if options.Header != nil {
		httpReq.Header = options.Header.Clone()
	}

	httpReq.Header.Set(headerUserAgent, userAgent)
	response, err := h.client.options.HTTPCaller(httpReq)
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
