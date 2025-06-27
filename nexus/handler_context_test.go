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
func (h *myHandler) StartOperation(ctx context.Context, service, operation string, input *nexus.LazyValue, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[any], error) {
	if err := h.authorize(ctx, options.Header); err != nil {
		return nil, err
	}
	return &nexus.HandlerStartOperationResultAsync{OperationToken: "some-token"}, nil
}

// getOperationResult implements the Handler interface.
func (h *myHandler) GetOperationResult(ctx context.Context, service, operation, token string, options nexus.GetOperationResultOptions) (any, error) {
	if err := h.authorize(ctx, options.Header); err != nil {
		return nil, err
	}
	if options.Wait > 0 { // request is a long poll
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Wait)
		defer cancel()

		result, err := h.pollOperation(ctx, options.Wait)
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
			return nil, &nexus.OperationError{
				State: nexus.OperationStateFailed,
				Cause: err,
			}
		}
		return result, nil
	} else {
		result, err := h.peekOperation(ctx)
		if err != nil {
			// Optionally translate to operation failure (could also result in canceled state).
			return nil, &nexus.OperationError{
				State: nexus.OperationStateFailed,
				Cause: err,
			}
		}
		return result, nil
	}
}

func (h *myHandler) CancelOperation(ctx context.Context, service, operation, token string, options nexus.CancelOperationOptions) error {
	// Handlers must implement this.
	panic("unimplemented")
}

func (h *myHandler) GetOperationInfo(ctx context.Context, service, operation, token string, options nexus.GetOperationInfoOptions) (*nexus.OperationInfo, error) {
	// Handlers must implement this.
	panic("unimplemented")
}

func (h *myHandler) pollOperation(ctx context.Context, wait time.Duration) (*MyResult, error) {
	panic("unimplemented")
}

func (h *myHandler) peekOperation(ctx context.Context) (*MyResult, error) {
	panic("unimplemented")
}

func (h *myHandler) authorize(_ context.Context, header nexus.Header) error {
	// Authorization for demo purposes
	if header.Get("Authorization") != "Bearer top-secret" {
		return nexus.HandlerErrorf(nexus.HandlerErrorTypeUnauthorized, "unauthorized")
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
