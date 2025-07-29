package nexus

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// CompletionRequest is input for CompletionHandler.CompleteOperation.
//
// NOTE: Experimental
type CompletionRequest struct {
	// The original HTTP request.
	HTTPRequest *http.Request
	// State of the operation.
	State OperationState
	// OperationID is the unique ID for this operation. Used when a completion callback is received before a started response.
	//
	// Deprecated: Use OperatonToken instead.
	OperationID string
	// OperationToken is the unique token for this operation. Used when a completion callback is received before a
	// started response.
	OperationToken string
	// StartTime is the time the operation started. Used when a completion callback is received before a started response.
	StartTime time.Time
	// Links are used to link back to the operation when a completion callback is received before a started response.
	Links []Link
	// Parsed from request and set if State is failed or canceled.
	Error error
	// Extracted from request and set if State is succeeded.
	Result *LazyValue
}

// A CompletionHandler can receive operation completion requests as delivered via the callback URL provided in
// start-operation requests.
//
// NOTE: Experimental
type CompletionHandler interface {
	CompleteOperation(context.Context, *CompletionRequest) error
}

// CompletionHandlerOptions are options for [NewCompletionHTTPHandler].
//
// NOTE: Experimental
type CompletionHandlerOptions struct {
	// Handler for completion requests.
	Handler CompletionHandler
	// A stuctured logging handler.
	// Defaults to slog.Default().
	Logger *slog.Logger
	// A [Serializer] to customize handler serialization behavior.
	// By default the handler handles, JSONables, byte slices, and nil.
	Serializer Serializer
	// A [FailureConverter] to convert a [Failure] instance to and from an [error]. Defaults to
	// [DefaultFailureConverter].
	FailureConverter FailureConverter
}

type completionHTTPHandler struct {
	baseHTTPHandler
	options CompletionHandlerOptions
}

func (h *completionHTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	completion := CompletionRequest{
		State:          OperationState(request.Header.Get(headerOperationState)),
		OperationID:    request.Header.Get(HeaderOperationID),
		OperationToken: request.Header.Get(HeaderOperationToken),
		HTTPRequest:    request,
	}
	if completion.OperationID == "" && completion.OperationToken != "" {
		completion.OperationID = completion.OperationToken
	} else if completion.OperationToken == "" && completion.OperationID != "" {
		completion.OperationToken = completion.OperationID
	}
	if startTimeHeader := request.Header.Get(headerOperationStartTime); startTimeHeader != "" {
		var parseTimeErr error
		if completion.StartTime, parseTimeErr = http.ParseTime(startTimeHeader); parseTimeErr != nil {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse operation start time header"))
			return
		}
	}
	var decodeErr error
	if completion.Links, decodeErr = getLinksFromHeader(request.Header); decodeErr != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to decode links from request headers"))
		return
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
		completion.Error = h.failureConverter.FailureToError(failure)
	case OperationStateSucceeded:
		completion.Result = &LazyValue{
			serializer: h.options.Serializer,
			Reader: &Reader{
				request.Body,
				prefixStrippedHTTPHeaderToNexusHeader(request.Header, "content-"),
			},
		}
	default:
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request operation state: %q", completion.State))
		return
	}
	if err := h.options.Handler.CompleteOperation(ctx, &completion); err != nil {
		h.writeFailure(writer, err)
	}
}

// NewCompletionHTTPHandler constructs an [http.Handler] from given options for handling operation completion requests.
//
// NOTE: Experimental
func NewCompletionHTTPHandler(options CompletionHandlerOptions) http.Handler {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.Serializer == nil {
		options.Serializer = defaultSerializer
	}
	if options.FailureConverter == nil {
		options.FailureConverter = defaultFailureConverter
	}
	return &completionHTTPHandler{
		options: options,
		baseHTTPHandler: baseHTTPHandler{
			logger:           options.Logger,
			failureConverter: options.FailureConverter,
		},
	}
}
