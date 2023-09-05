package nexus_test

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
)

type myHandler struct {
	nexus.UnimplementedHandler
}

type MyResult struct {
	Field string `json:"field"`
}

// StartOperation implements the Handler interface.
func (h *myHandler) StartOperation(ctx context.Context, request *nexus.StartOperationRequest) (nexus.OperationResponse, error) {
	if err := h.authorize(ctx, request.HTTPRequest); err != nil {
		return nil, err
	}
	return &nexus.OperationResponseAsync{
		OperationID: "TODO",
	}, nil
}

// GetOperationResult implements the Handler interface.
func (h *myHandler) GetOperationResult(ctx context.Context, request *nexus.GetOperationResultRequest) (*nexus.OperationResponseSync, error) {
	if err := h.authorize(ctx, request.HTTPRequest); err != nil {
		return nil, err
	}
	if request.Wait > 0 { // request is a long poll
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Wait)
		defer cancel()

		result, err := h.pollOperation(ctx, request.Wait)
		if err != nil {
			// Translate deadline exceeded to "OperationStillRunning", this may or may not be semantically correct for
			// your application.
			// Some applications may want to "peek" the current status instead of performing this blind conversion if
			// the wait time is exceeded and the request's context deadline has not yet exceeded.
			if ctx.Err() != nil {
				return nil, nexus.ErrOperationStillRunning
			}
			// Optionally translate to operation failure (could also result in canceled state).
			// Optionally expose the error details to the caller.
			return nil, &nexus.UnsuccessfulOperationError{State: nexus.OperationStateFailed, Failure: nexus.Failure{Message: err.Error()}}
		}
		return nexus.NewOperationResponseSync(result)
	} else {
		result, err := h.peekOperation(ctx)
		if err != nil {
			// Optionally translate to operation failure (could also result in canceled state).
			return nil, &nexus.UnsuccessfulOperationError{State: nexus.OperationStateFailed, Failure: nexus.Failure{Message: err.Error()}}
		}
		return nexus.NewOperationResponseSync(result)
	}
}

func (h *myHandler) CancelOperation(ctx context.Context, request *nexus.CancelOperationRequest) error {
	// Handlers must implement this.
	panic("unimplemented")
}

func (h *myHandler) GetOperationInfo(ctx context.Context, request *nexus.GetOperationInfoRequest) (*nexus.OperationInfo, error) {
	// Handlers must implement this.
	panic("unimplemented")
}

func (h *myHandler) pollOperation(ctx context.Context, wait time.Duration) (*MyResult, error) {
	panic("unimplemented")
}

func (h *myHandler) peekOperation(ctx context.Context) (*MyResult, error) {
	panic("unimplemented")
}

func (h *myHandler) authorize(ctx context.Context, request *http.Request) error {
	// Authorization for demo purposes
	if request.Header.Get("Authorization") != "Bearer top-secret" {
		return &nexus.HandlerError{StatusCode: http.StatusUnauthorized, Failure: &nexus.Failure{Message: "Unauthorized"}}
	}
	return nil
}

func ExampleHandler() {
	handler := &myHandler{}
	httpHandler := nexus.NewHTTPHandler(nexus.HandlerOptions{Handler: handler})

	listener, _ := net.Listen("tcp", "localhost:0")
	defer listener.Close()
	_ = http.Serve(listener, httpHandler)
}
