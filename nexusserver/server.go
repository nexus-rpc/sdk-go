package nexusserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	"github.com/nexus-rpc/sdk-go/nexusapi"
	"golang.org/x/exp/slog"
)

type (
	Marshaler = func(v any) ([]byte, error)

	Options struct {
		Handler    Handler
		Marshaler  Marshaler
		LogHandler slog.Handler
	}

	ResultWriter interface {
		Header() http.Header
		Write([]byte) (int, error)
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

	AsyncOperation struct {
		ID string
	}

	Handler interface {
		StartOperation(context.Context, ResultWriter, *StartOperationRequest) error
		GetOperationResult(context.Context, ResultWriter, *GetOperationResultRequest) error
		GetOperationInfo(context.Context, *GetOperationInfoRequest) (nexusapi.OperationInfo, error)
		CancelOperation(context.Context, *CancelOperationRequest) error
	}

	httpHandler struct {
		options Options
		logger  slog.Logger
	}

	resultWriter struct {
		writeCalled bool
		httpWriter  http.ResponseWriter
	}
)

const (
	HeaderOperationState = nexusapi.HeaderOperationState
	HeaderContentType    = nexusapi.HeaderContentType
	ContentTypeJSON      = nexusapi.ContentTypeJSON
)

var ErrInternal = errors.New("internal server error")

func (w *resultWriter) Header() http.Header {
	return w.httpWriter.Header()
}

func (w *resultWriter) Write(b []byte) (int, error) {
	w.writeCalled = true
	return w.httpWriter.Write(b)
}

func (o AsyncOperation) Error() string {
	return "async operation"
}

func (h *httpHandler) WriteFailure(writer http.ResponseWriter, err error) {
	var failure nexusapi.Failure
	var unsuccessfulError *nexusapi.UnsuccessfulOperationError
	var operationState nexusapi.OperationState

	if errors.As(err, &unsuccessfulError) {
		operationState = unsuccessfulError.State
		failure = unsuccessfulError.Failure
	} else {
		failure = nexusapi.Failure{
			Message: err.Error(),
		}
	}

	bytes, err := h.options.Marshaler(failure)
	if err != nil {
		h.logger.Error("failed to marshal failure", "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set(HeaderContentType, ContentTypeJSON)

	if operationState == nexusapi.OperationStateFailed || operationState == nexusapi.OperationStateCanceled {
		writer.Header().Set(HeaderOperationState, string(operationState))
		writer.WriteHeader(nexusapi.StatusOperationFailed)
	} else {
		h.logger.Error("unexpected operation state", "state", operationState)
		writer.WriteHeader(http.StatusInternalServerError)
	}
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
	rw := resultWriter{httpWriter: writer}

	if err := h.options.Handler.StartOperation(request.Context(), &rw, handlerRequest); err != nil {
		if rw.writeCalled {
			h.logger.Error("ignoring error because handler wrote response body", "error", err)
			return
		}
		var async *AsyncOperation
		if errors.As(err, &async) {
			info := nexusapi.OperationInfo{
				ID:    async.ID,
				State: nexusapi.OperationStateRunning,
			}
			bytes, err := h.options.Marshaler(info)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}

			writer.Header().Set(HeaderContentType, ContentTypeJSON)
			writer.WriteHeader(http.StatusCreated)

			if _, err := writer.Write(bytes); err != nil {
				h.logger.Error("failed to write response body", "error", err)
			}
			return
		}
		h.WriteFailure(writer, err)
	}
}

func (h *httpHandler) GetOperationResult(writer http.ResponseWriter, request *http.Request) {
	// strip /result
	prefix, operationID := path.Split(path.Dir(request.URL.Path))
	operation := path.Base(prefix)
	handlerRequest := &GetOperationResultRequest{Operation: operation, OperationID: operationID, HTTPRequest: request}
	rw := resultWriter{httpWriter: writer}

	if err := h.options.Handler.GetOperationResult(request.Context(), &rw, handlerRequest); err != nil {
		if rw.writeCalled {
			h.logger.Error("ignoring error because handler wrote response body", "error", err)
			return
		}
		h.WriteFailure(writer, err)
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
		h.logger.Error("failed to marshal operation info", "error", err)
		h.WriteFailure(writer, ErrInternal)
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
		options.Marshaler = func(v any) ([]byte, error) {
			return json.MarshalIndent(v, "", "  ")
		}
	}
	if options.LogHandler == nil {
		options.LogHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	handler := &httpHandler{options, *slog.New(options.LogHandler)}
	router := mux.NewRouter()
	router.HandleFunc("/{operation}", handler.StartOperation).Methods("POST")
	router.HandleFunc("/{operation}/{operation_id}", handler.GetOperationInfo).Methods("GET")
	router.HandleFunc("/{operation}/{operation_id}/result", handler.GetOperationResult).Methods("GET")
	router.HandleFunc("/{operation}/{operation_id}/cancel", handler.CancelOperation).Methods("POST")
	return router
}
