package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// An HandlerStartOperationResult is the return type from the [Handler] StartOperation and [Operation] Start methods. It
// has two implementations: [HandlerStartOperationResultSync] and [HandlerStartOperationResultAsync].
type HandlerStartOperationResult[T any] interface {
	applyToHTTPResponse(http.ResponseWriter, *httpHandler)
}

// HandlerStartOperationResultSync indicates that an operation completed successfully.
type HandlerStartOperationResultSync[T any] struct {
	Value T
}

func (r *HandlerStartOperationResultSync[T]) applyToHTTPResponse(writer http.ResponseWriter, handler *httpHandler) {
	handler.writeResult(writer, r.Value)
}

// HandlerStartOperationResultAsync indicates that an operation has been accepted and will complete asynchronously.
type HandlerStartOperationResultAsync struct {
	OperationID string
}

func (r *HandlerStartOperationResultAsync) applyToHTTPResponse(writer http.ResponseWriter, handler *httpHandler) {
	info := OperationInfo{
		ID:    r.OperationID,
		State: OperationStateRunning,
	}
	bytes, err := json.Marshal(info)
	if err != nil {
		handler.logger.Error("failed to serialize operation info", "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", contentTypeJSON)
	writer.WriteHeader(http.StatusCreated)

	if _, err := writer.Write(bytes); err != nil {
		handler.logger.Error("failed to write response body", "error", err)
	}
}

// A Handler must implement all of the Nexus service endpoints as defined in the [Nexus HTTP API].
//
// Handler implementations must embed the [UnimplementedHandler].
//
// All Handler methods can return a [HandlerError] to fail requests with a custom [HandlerErrorType] and structured [Failure].
// Arbitrary errors from handler methods are turned into [HandlerErrorTypeInternal],their details are logged and hidden
// from the caller.
//
// [Nexus HTTP API]: https://github.com/nexus-rpc/api
type Handler interface {
	// StartOperation handles requests for starting an operation. Return [HandlerStartOperationResultSync] to
	// respond successfully - inline, or [HandlerStartOperationResultAsync] to indicate that an asynchronous
	// operation was started. Return an [UnsuccessfulOperationError] to indicate that an operation completed as
	// failed or canceled.
	StartOperation(ctx context.Context, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error)
	// GetOperationResult handles requests to get the result of an asynchronous operation. Return non error result
	// to respond successfully - inline, or error with [ErrOperationStillRunning] to indicate that an asynchronous
	// operation is still running. Return an [UnsuccessfulOperationError] to indicate that an operation completed as
	// failed or canceled.
	//
	// When [GetOperationResultOptions.Wait] is greater than zero, this request should be treated as a long poll.
	// Long poll requests have a server side timeout, configurable via [HandlerOptions.GetResultTimeout], and exposed
	// via context deadline. The context deadline is decoupled from the application level Wait duration.
	//
	// It is the implementor's responsiblity to respect the client's wait duration and return in a timely fashion.
	// Consider using a derived context that enforces the wait timeout when implementing this method and return
	// [ErrOperationStillRunning] when that context expires as shown in the example.
	GetOperationResult(ctx context.Context, operation, operationID string, options GetOperationResultOptions) (any, error)
	// GetOperationInfo handles requests to get information about an asynchronous operation.
	GetOperationInfo(ctx context.Context, operation, operationID string, options GetOperationInfoOptions) (*OperationInfo, error)
	// CancelOperation handles requests to cancel an asynchronous operation.
	// Cancelation in Nexus is:
	//  1. asynchronous - returning from this method only ensures that cancelation is delivered, it may later be
	//  ignored by the underlying operation implemention.
	//  2. idempotent - implementors should ignore duplicate cancelations for the same operation.
	CancelOperation(ctx context.Context, operation, operationID string, options CancelOperationOptions) error
	mustEmbedUnimplementedHandler()
}

type HandlerErrorType string

const (
	// An internal error occured.
	HandlerErrorTypeInternal HandlerErrorType = "INTERNAL"
	// Used by gateways to report that a downstream server has responded with an error.
	HandlerErrorTypeUpstreamError HandlerErrorType = "DOWNSTREAM_ERROR"
	// Used by gateways to report that a request to a downstream server has timed out.
	HandlerErrorTypeDownstreamTimeout HandlerErrorType = "DOWNSTREAM_TIMEOUT"
	// The client did not supply valid authentication credentials for this request.
	HandlerErrorTypeUnauthenticated HandlerErrorType = "UNAUTHENTICATED"
	// The caller does not have permission to execute the specified operation.
	HandlerErrorTypeForbidden HandlerErrorType = "UNAUTHORIZED"
	// The server cannot or will not process the request due to an apparent client error.
	HandlerErrorTypeBadRequest HandlerErrorType = "BAD_REQUEST"
	// The requested resource could not be found but may be available in the future. Subsequent requests by the client
	// are permissible.
	HandlerErrorTypeNotFound HandlerErrorType = "NOT_FOUND"
	// The server either does not recognize the request method, or it lacks the ability to fulfill the request.
	HandlerErrorTypeNotImplemented HandlerErrorType = "NOT_IMPLEMENTED"
)

// HandlerError is a special error that can be returned from [Handler] methods for failing a request with a custom
// status code and failure message.
type HandlerError struct {
	// Defaults to HandlerErrorTypeInternal
	Type HandlerErrorType
	// Failure to report back in the response. Optional.
	Failure *Failure
}

// Error implements the error interface.
func (e *HandlerError) Error() string {
	typ := e.Type
	if len(typ) == 0 {
		typ = HandlerErrorTypeInternal
	}
	if e.Failure != nil {
		return fmt.Sprintf("handler error (%s): %s", typ, e.Failure.Message)
	}
	return fmt.Sprintf("handler error (%s)", typ)
}

// HandlerErrorf creates a [HandlerError] with the given type and a formatted failure message.
func HandlerErrorf(typ HandlerErrorType, format string, args ...any) *HandlerError {
	return &HandlerError{
		Type: typ,
		Failure: &Failure{
			Message: fmt.Sprintf(format, args...),
		},
	}
}

type baseHTTPHandler struct {
	logger *slog.Logger
}

type httpHandler struct {
	baseHTTPHandler
	options HandlerOptions
}

func (h *httpHandler) writeResult(writer http.ResponseWriter, result any) {
	var reader *Reader
	if r, ok := result.(*Reader); ok {
		// Close the request body in case we error before sending the HTTP request (which may double close but
		// that's fine since we ignore the error).
		defer r.Close()
		reader = r
	} else {
		content, ok := result.(*Content)
		if !ok {
			var err error
			content, err = h.options.Serializer.Serialize(result)
			if err != nil {
				h.writeFailure(writer, fmt.Errorf("failed to serialize handler result: %w", err))
				return
			}
		}
		header := maps.Clone(content.Header)
		header["length"] = strconv.Itoa(len(content.Data))

		reader = &Reader{
			io.NopCloser(bytes.NewReader(content.Data)),
			header,
		}
	}

	header := writer.Header()
	addContentHeaderToHTTPHeader(reader.Header, header)
	if reader.ReadCloser == nil {
		return
	}
	if _, err := io.Copy(writer, reader); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *baseHTTPHandler) writeFailure(writer http.ResponseWriter, err error) {
	var failure *Failure
	var unsuccessfulError *UnsuccessfulOperationError
	var handlerError *HandlerError
	var operationState OperationState
	statusCode := http.StatusInternalServerError

	if errors.As(err, &unsuccessfulError) {
		operationState = unsuccessfulError.State
		failure = &unsuccessfulError.Failure
		statusCode = statusOperationFailed

		if operationState == OperationStateFailed || operationState == OperationStateCanceled {
			writer.Header().Set(headerOperationState, string(operationState))
		} else {
			h.logger.Error("unexpected operation state", "state", operationState)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if errors.As(err, &handlerError) {
		failure = handlerError.Failure
		switch handlerError.Type {
		case HandlerErrorTypeDownstreamTimeout:
			statusCode = statusDownstreamTimeout
		case HandlerErrorTypeUpstreamError:
			statusCode = statusDownstreamError
		case HandlerErrorTypeBadRequest:
			statusCode = http.StatusBadRequest
		case HandlerErrorTypeForbidden:
			statusCode = http.StatusForbidden
		case HandlerErrorTypeUnauthenticated:
			statusCode = http.StatusUnauthorized
		case HandlerErrorTypeNotFound:
			statusCode = http.StatusNotFound
		case HandlerErrorTypeNotImplemented:
			statusCode = http.StatusNotImplemented
		case HandlerErrorTypeInternal:
			statusCode = http.StatusInternalServerError
		default:
			h.logger.Error("unexpected handler error type", "type", handlerError.Type)
		}
	} else {
		failure = &Failure{
			Message: "internal server error",
		}
		h.logger.Error("handler failed", "error", err)
	}

	var bytes []byte
	if failure != nil {
		bytes, err = json.Marshal(failure)
		if err != nil {
			h.logger.Error("failed to marshal failure", "error", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", contentTypeJSON)
	}

	writer.WriteHeader(statusCode)

	if _, err := writer.Write(bytes); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *httpHandler) startOperation(writer http.ResponseWriter, request *http.Request) {
	operation, err := url.PathUnescape(path.Base(request.URL.EscapedPath()))
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	options := StartOperationOptions{
		RequestID:   request.Header.Get(headerRequestID),
		CallbackURL: request.URL.Query().Get(queryCallbackURL),
		Header:      httpHeaderToNexusHeader(request.Header),
	}
	value := &LazyValue{
		serializer: h.options.Serializer,
		Reader: &Reader{
			request.Body,
			httpHeaderToContentHeader(request.Header),
		},
	}
	response, err := h.options.Handler.StartOperation(request.Context(), operation, value, options)
	if err != nil {
		h.writeFailure(writer, err)
	} else {
		response.applyToHTTPResponse(writer, h)
	}
}

func (h *httpHandler) getOperationResult(writer http.ResponseWriter, request *http.Request) {
	// strip /result
	prefix, operationIDEscaped := path.Split(path.Dir(request.URL.EscapedPath()))
	operationID, err := url.PathUnescape(operationIDEscaped)
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	operation, err := url.PathUnescape(path.Base(prefix))
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	options := GetOperationResultOptions{Header: httpHeaderToNexusHeader(request.Header)}

	waitStr := request.URL.Query().Get(queryWait)
	ctx := request.Context()
	if waitStr != "" {
		waitDuration, err := time.ParseDuration(waitStr)
		if err != nil {
			h.logger.Warn("invalid wait duration query parameter", "wait", waitStr)
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid wait query parameter"))
			return
		}
		options.Wait = waitDuration
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(request.Context(), h.options.GetResultTimeout)
		defer cancel()
	}

	result, err := h.options.Handler.GetOperationResult(ctx, operation, operationID, options)
	if err != nil {
		if options.Wait > 0 && ctx.Err() != nil {
			writer.WriteHeader(http.StatusRequestTimeout)
		} else if errors.Is(err, ErrOperationStillRunning) {
			writer.WriteHeader(statusOperationRunning)
		} else {
			h.writeFailure(writer, err)
		}
		return
	}
	h.writeResult(writer, result)
}

func (h *httpHandler) getOperationInfo(writer http.ResponseWriter, request *http.Request) {
	prefix, operationIDEscaped := path.Split(request.URL.EscapedPath())
	operationID, err := url.PathUnescape(operationIDEscaped)
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	operation, err := url.PathUnescape(path.Base(prefix))
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	options := GetOperationInfoOptions{Header: httpHeaderToNexusHeader(request.Header)}

	info, err := h.options.Handler.GetOperationInfo(request.Context(), operation, operationID, options)
	if err != nil {
		h.writeFailure(writer, err)
		return
	}

	bytes, err := json.Marshal(info)
	if err != nil {
		h.writeFailure(writer, fmt.Errorf("failed to marshal operation info: %w", err))
		return
	}
	writer.Header().Set("Content-Type", contentTypeJSON)
	if _, err := writer.Write(bytes); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *httpHandler) cancelOperation(writer http.ResponseWriter, request *http.Request) {
	// strip /cancel
	prefix, operationIDEscaped := path.Split(path.Dir(request.URL.EscapedPath()))
	operationID, err := url.PathUnescape(operationIDEscaped)
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	operation, err := url.PathUnescape(path.Base(prefix))
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	options := CancelOperationOptions{Header: httpHeaderToNexusHeader(request.Header)}

	if err := h.options.Handler.CancelOperation(request.Context(), operation, operationID, options); err != nil {
		h.writeFailure(writer, err)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

// HandlerOptions are options for [NewHTTPHandler].
type HandlerOptions struct {
	// Handler for handling service requests.
	Handler Handler
	// A stuctured logger.
	// Defaults to slog.Default().
	Logger *slog.Logger
	// Max duration to allow waiting for a single get result request.
	// Enforced if provided for requests with the wait query parameter set.
	//
	// Defaults to one minute.
	GetResultTimeout time.Duration
	// A [Serializer] to customize handler serialization behavior.
	// By default the handler handles, JSONables, byte slices, and nil.
	Serializer Serializer
}

// NewHTTPHandler constructs an [http.Handler] from given options for handling Nexus service requests.
func NewHTTPHandler(options HandlerOptions) http.Handler {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.GetResultTimeout == 0 {
		options.GetResultTimeout = time.Minute
	}
	if options.Serializer == nil {
		options.Serializer = defaultSerializer
	}
	handler := &httpHandler{
		baseHTTPHandler: baseHTTPHandler{
			logger: slog.Default(),
		},
		options: options,
	}

	router := mux.NewRouter().UseEncodedPath()
	router.HandleFunc("/{operation}", handler.startOperation).Methods("POST")
	router.HandleFunc("/{operation}/{operation_id}", handler.getOperationInfo).Methods("GET")
	router.HandleFunc("/{operation}/{operation_id}/result", handler.getOperationResult).Methods("GET")
	router.HandleFunc("/{operation}/{operation_id}/cancel", handler.cancelOperation).Methods("POST")
	return router
}
