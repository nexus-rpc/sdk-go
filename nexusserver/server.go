package nexusserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/nexus-rpc/sdk-go/nexusapi"
	"golang.org/x/exp/slog"
)

type (
	Marshaler = func(v any) ([]byte, error)

	Options struct {
		Handler                    Handler
		Marshaler                  Marshaler
		LogHandler                 slog.Handler
		GetResultMaxRequestTimeout time.Duration
	}

	StartOperationRequest struct {
		Operation   string
		RequestID   string
		CallbackURL string
		HTTPRequest *http.Request
	}

	GetOperationResultRequest struct {
		Operation   string
		OperationID string
		Wait        bool
		HTTPRequest *http.Request
	}

	GetOperationInfoRequest struct {
		Operation   string
		OperationID string
		HTTPRequest *http.Request
	}

	CancelOperationRequest struct {
		Operation   string
		OperationID string
		HTTPRequest *http.Request
	}

	OperationResponse interface {
		applyToStartResponse(http.ResponseWriter, *httpHandler)
		applyToGetResultResponse(http.ResponseWriter, *httpHandler)
	}

	HandlerError struct {
		// Defaults to 500
		StatusCode int
		Failure    *nexusapi.Failure
	}

	Handler interface {
		StartOperation(context.Context, *StartOperationRequest) (OperationResponse, error)
		GetOperationResult(context.Context, *GetOperationResultRequest) (OperationResponse, error)
		GetOperationInfo(context.Context, *GetOperationInfoRequest) (*nexusapi.OperationInfo, error)
		CancelOperation(context.Context, *CancelOperationRequest) error
	}

	baseHTTPHandler struct {
		marshaler Marshaler
		logger    *slog.Logger
	}

	httpHandler struct {
		baseHTTPHandler
		options Options
	}

	OperationResponseSync struct {
		Header  http.Header
		Content io.ReadCloser
	}

	OperationResponseAsync struct {
		Header      http.Header
		OperationID string
	}
)

const (
	HeaderOperationState = nexusapi.HeaderOperationState
	HeaderContentType    = nexusapi.HeaderContentType
	ContentTypeJSON      = nexusapi.ContentTypeJSON
)

func newBadRequestError(format string, args ...any) *HandlerError {
	return &HandlerError{
		StatusCode: http.StatusBadRequest,
		Failure: &nexusapi.Failure{
			Message: fmt.Sprintf(format, args...),
		},
	}
}

func (e *HandlerError) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.Failure.Message)
}

func (r *OperationResponseAsync) applyToStartResponse(writer http.ResponseWriter, handler *httpHandler) {
	info := nexusapi.OperationInfo{
		ID:    r.OperationID,
		State: nexusapi.OperationStateRunning,
	}
	bytes, err := handler.marshaler(info)
	if err != nil {
		handler.logger.Error("failed to serialize operation info", "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set(HeaderContentType, ContentTypeJSON)
	writer.WriteHeader(http.StatusCreated)

	if _, err := writer.Write(bytes); err != nil {
		handler.logger.Error("failed to write response body", "error", err)
	}
}

func (r *OperationResponseAsync) applyToGetResultResponse(writer http.ResponseWriter, handler *httpHandler) {
	writer.Header().Set(HeaderOperationState, string(nexusapi.OperationStateRunning))
	writer.WriteHeader(http.StatusNoContent)
}

func (r *OperationResponseSync) applyToStartResponse(writer http.ResponseWriter, handler *httpHandler) {
	header := writer.Header()
	for k, v := range r.Header {
		header[k] = v
	}
	defer r.Content.Close()
	if _, err := io.Copy(writer, r.Content); err != nil {
		handler.logger.Error("failed to write response body", "error", err)
	}
}

func (r *OperationResponseSync) applyToGetResultResponse(writer http.ResponseWriter, handler *httpHandler) {
	writer.Header().Set(HeaderOperationState, string(nexusapi.OperationStateSucceeded))
	r.applyToStartResponse(writer, handler)
}

func NewBytesOperationResultSync(header http.Header, b []byte) (*OperationResponseSync, error) {
	return &OperationResponseSync{
		Header:  header,
		Content: io.NopCloser(bytes.NewReader(b)),
	}, nil
}

func NewJSONOperationResultSync(header http.Header, v any) (*OperationResponseSync, error) {
	// TODO: do we want an indent option too?
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	header = header.Clone()
	if header == nil {
		header = make(http.Header)
	}
	header.Set(HeaderContentType, ContentTypeJSON)
	return &OperationResponseSync{
		Header:  header,
		Content: io.NopCloser(bytes.NewReader(b)),
	}, nil
}

// TODO: test me
func (h *baseHTTPHandler) WriteFailure(writer http.ResponseWriter, err error) {
	var failure *nexusapi.Failure
	var unsuccessfulError *nexusapi.UnsuccessfulOperationError
	var handlerError *HandlerError
	var operationState nexusapi.OperationState
	statusCode := http.StatusInternalServerError

	if errors.As(err, &unsuccessfulError) {
		operationState = unsuccessfulError.State
		failure = unsuccessfulError.Failure
		statusCode = nexusapi.StatusOperationFailed

		if operationState == nexusapi.OperationStateFailed || operationState == nexusapi.OperationStateCanceled {
			writer.Header().Set(HeaderOperationState, string(operationState))
		} else {
			h.logger.Error("unexpected operation state", "state", operationState)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if errors.As(err, &handlerError) {
		failure = handlerError.Failure
		statusCode = handlerError.StatusCode
	} else {
		h.logger.Error("handler failed", "error", err)
	}

	var bytes []byte
	if failure != nil {
		bytes, err = h.marshaler(failure)
		if err != nil {
			h.logger.Error("failed to marshal failure", "error", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Header().Set(HeaderContentType, ContentTypeJSON)
	}

	writer.WriteHeader(statusCode)

	if _, err := writer.Write(bytes); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *httpHandler) StartOperation(writer http.ResponseWriter, request *http.Request) {
	operation := path.Base(request.URL.Path)
	handlerRequest := &StartOperationRequest{
		Operation:   operation,
		RequestID:   request.Header.Get(nexusapi.HeaderRequestID),
		CallbackURL: request.URL.Query().Get(nexusapi.QueryCallbackURL),
		HTTPRequest: request,
	}
	response, err := h.options.Handler.StartOperation(request.Context(), handlerRequest)
	if err != nil {
		h.WriteFailure(writer, err)
	} else {
		response.applyToStartResponse(writer, h)
	}
}

func (h *httpHandler) GetOperationResult(writer http.ResponseWriter, request *http.Request) {
	// strip /result
	prefix, operationID := path.Split(path.Dir(request.URL.Path))
	operation := path.Base(prefix)
	handlerRequest := &GetOperationResultRequest{Operation: operation, OperationID: operationID, HTTPRequest: request}

	ctx := request.Context()
	waitStr := request.URL.Query().Get(nexusapi.QueryWait)
	if waitStr != "" {
		waitDuration, err := time.ParseDuration(waitStr)
		if err != nil {
			h.logger.Warn("invalid wait duration query parameter", "wait", waitStr)
			h.WriteFailure(writer, newBadRequestError("invalid wait query parameter"))
			return
		}

		var cancel func()
		if waitDuration > h.options.GetResultMaxRequestTimeout {
			waitDuration = h.options.GetResultMaxRequestTimeout
		}
		// TODO: reduce duration a bit to give some grace time?
		ctx, cancel = context.WithTimeout(ctx, waitDuration)
		handlerRequest.Wait = true
		defer cancel()
	}

	response, err := h.options.Handler.GetOperationResult(ctx, handlerRequest)
	if err != nil {
		h.WriteFailure(writer, err)
	} else {
		response.applyToGetResultResponse(writer, h)
	}
}

func (h *httpHandler) GetOperationInfo(writer http.ResponseWriter, request *http.Request) {
	prefix, operationID := path.Split(request.URL.Path)
	operation := path.Base(prefix)
	handlerRequest := &GetOperationInfoRequest{Operation: operation, OperationID: operationID, HTTPRequest: request}

	info, err := h.options.Handler.GetOperationInfo(request.Context(), handlerRequest)
	if err != nil {
		h.WriteFailure(writer, err)
		return
	}

	bytes, err := h.options.Marshaler(info)
	if err != nil {
		h.WriteFailure(writer, fmt.Errorf("failed to marshal operation info: %w", err))
		return
	}
	writer.Header().Set(HeaderContentType, ContentTypeJSON)
	if _, err := writer.Write(bytes); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *httpHandler) CancelOperation(writer http.ResponseWriter, request *http.Request) {
	// strip /cancel
	prefix, operationID := path.Split(path.Dir(request.URL.Path))
	operation := path.Base(prefix)
	handlerRequest := &CancelOperationRequest{Operation: operation, OperationID: operationID, HTTPRequest: request}

	if err := h.options.Handler.CancelOperation(request.Context(), handlerRequest); err != nil {
		h.WriteFailure(writer, err)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

func NewHTTPHandler(options Options) http.Handler {
	if options.Marshaler == nil {
		options.Marshaler = defaultMarshaller
	}
	if options.LogHandler == nil {
		options.LogHandler = newDefaultLogHandler()
	}
	if options.GetResultMaxRequestTimeout == 0 {
		options.GetResultMaxRequestTimeout = time.Minute
	}
	handler := &httpHandler{
		baseHTTPHandler: baseHTTPHandler{
			logger:    slog.New(options.LogHandler),
			marshaler: options.Marshaler,
		},
		options: options,
	}

	router := mux.NewRouter()
	router.HandleFunc("/{operation}", handler.StartOperation).Methods("POST")
	router.HandleFunc("/{operation}/{operation_id}", handler.GetOperationInfo).Methods("GET")
	router.HandleFunc("/{operation}/{operation_id}/result", handler.GetOperationResult).Methods("GET")
	router.HandleFunc("/{operation}/{operation_id}/cancel", handler.CancelOperation).Methods("POST")
	return router
}

type CompletionRequest struct {
	// The original HTTP request.
	HTTPRequest *http.Request
	// State of the operation.
	State nexusapi.OperationState
	// Parsed from request and set if State is not failed or canceled.
	Failure *nexusapi.Failure
}

type CompletionHandler interface {
	Complete(context.Context, *CompletionRequest) error
}

type CompletionOptions struct {
	Handler    CompletionHandler
	LogHandler slog.Handler
	Marshaler  Marshaler
}

type completionHTTPHandler struct {
	baseHTTPHandler
	handler CompletionHandler
}

func (h *completionHTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	completion := CompletionRequest{
		State:       nexusapi.OperationState(request.Header.Get(HeaderOperationState)),
		HTTPRequest: request,
	}
	switch completion.State {
	case nexusapi.OperationStateFailed, nexusapi.OperationStateCanceled:
		if !nexusapi.IsContentTypeJSON(request.Header) {
			h.WriteFailure(writer, newBadRequestError("invalid request content type: %q", request.Header.Get(HeaderContentType)))
			return
		}
		var failure nexusapi.Failure
		b, err := io.ReadAll(request.Body)
		if err != nil {
			h.WriteFailure(writer, newBadRequestError("failed to read Failure from request body"))
			return
		}
		if err := json.Unmarshal(b, &failure); err != nil {
			h.WriteFailure(writer, newBadRequestError("failed to read Failure from request body"))
			return
		}
		completion.Failure = &failure
	case nexusapi.OperationStateSucceeded:
		// Nothing to do here.
	default:
		h.WriteFailure(writer, newBadRequestError("invalid request operation state: %q", completion.State))
		return
	}
	if err := h.handler.Complete(ctx, &completion); err != nil {
		h.WriteFailure(writer, err)
	}
}

func NewCompletionHTTPHandler(options CompletionOptions) http.Handler {
	if options.Marshaler == nil {
		options.Marshaler = defaultMarshaller
	}
	if options.LogHandler == nil {
		options.LogHandler = newDefaultLogHandler()
	}
	return &completionHTTPHandler{
		baseHTTPHandler: baseHTTPHandler{
			logger:    slog.New(options.LogHandler),
			marshaler: options.Marshaler,
		},
		handler: options.Handler,
	}
}

func defaultMarshaller(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func newDefaultLogHandler() slog.Handler {
	return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
}
