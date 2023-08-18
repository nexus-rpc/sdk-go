package nexusserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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
		applyToHTTP(http.ResponseWriter, Marshaler, slog.Logger)
	}

	Handler interface {
		StartOperation(context.Context, *StartOperationRequest) (OperationResponse, error)
		GetOperationResult(context.Context, *GetOperationResultRequest) (OperationResponse, error)
		GetOperationInfo(context.Context, *GetOperationInfoRequest) (nexusapi.OperationInfo, error)
		CancelOperation(context.Context, *CancelOperationRequest) error
	}

	httpHandler struct {
		options Options
		logger  slog.Logger
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

var ErrInternal = errors.New("internal server error")

func (response *OperationResponseAsync) applyToHTTP(writer http.ResponseWriter, marshaler Marshaler, logger slog.Logger) {
	info := nexusapi.OperationInfo{
		ID:    response.OperationID,
		State: nexusapi.OperationStateRunning,
	}
	bytes, err := marshaler(info)
	if err != nil {
		logger.Error("failed to serialize operation info", "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set(HeaderContentType, ContentTypeJSON)
	writer.WriteHeader(http.StatusCreated)

	if _, err := writer.Write(bytes); err != nil {
		logger.Error("failed to write response body", "error", err)
	}
}

func (response *OperationResponseSync) applyToHTTP(writer http.ResponseWriter, marshaler Marshaler, logger slog.Logger) {
	header := writer.Header()
	for k, v := range response.Header {
		header[k] = v
	}
	defer response.Content.Close()
	if _, err := io.Copy(writer, response.Content); err != nil {
		logger.Error("failed to write response body", "error", err)
	}
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
	response, err := h.options.Handler.StartOperation(request.Context(), handlerRequest)
	if err != nil {
		h.WriteFailure(writer, err)
	} else {
		response.applyToHTTP(writer, h.options.Marshaler, h.logger)
	}
}

func (h *httpHandler) GetOperationResult(writer http.ResponseWriter, request *http.Request) {
	// strip /result
	prefix, operationID := path.Split(path.Dir(request.URL.Path))
	operation := path.Base(prefix)
	handlerRequest := &GetOperationResultRequest{Operation: operation, OperationID: operationID, HTTPRequest: request}

	response, err := h.options.Handler.GetOperationResult(request.Context(), handlerRequest)
	if err != nil {
		h.WriteFailure(writer, err)
	} else {
		response.applyToHTTP(writer, h.options.Marshaler, h.logger)
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
