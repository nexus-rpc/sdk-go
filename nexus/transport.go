package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const getResultContextPadding = time.Second * 5

var errOperationWaitTimeout = errors.New("operation wait timeout")

type (
	Transport interface {
		StartOperation(
			ctx context.Context,
			service string,
			operation string,
			input any,
			options StartOperationOptions,
		) (*StartOperationResponse[*LazyValue], error)

		GetOperationInfo(
			ctx context.Context,
			service string,
			operation string,
			token string,
			options GetOperationInfoOptions,
		) (*GetOperationInfoResponse, error)

		GetOperationResult(
			ctx context.Context,
			service string,
			operation string,
			token string,
			options GetOperationResultOptions,
		) (*GetOperationResultResponse[*LazyValue], error)

		CancelOperation(
			ctx context.Context,
			service string,
			operation string,
			token string,
			options CancelOperationOptions,
		) (*CancelOperationResponse, error)
	}

	StartOperationResponse[T any] struct {
		// Set when start completes synchronously.
		//
		// If T is a [LazyValue], ensure that your consume it or read the underlying content in its entirety and close it to
		// free up the underlying connection.
		Complete *OperationResult[T]
		// Set when the handler indicates that it started an asynchronous operation.
		// The attached handle can be used to perform actions such as cancel the operation or get its result.
		Pending *OperationHandle[T]
		// Links contain information about the operations done by the handler.
		Links []Link
	}

	GetOperationInfoResponse struct {
		Info *OperationInfo
	}

	GetOperationResultResponse[T any] struct {
		result *OperationResult[T]
		Links  []Link
	}

	CancelOperationResponse struct {
	}

	OperationResult[T any] struct {
		result T
		err    error
	}
)

func (r *OperationResult[T]) Get() (T, error) {
	return r.result, r.err
}

func (gr *GetOperationResultResponse[T]) GetResult() (T, error) {
	return gr.result.Get()
}

// User-Agent header set on HTTP requests.
const userAgent = "Nexus-go-sdk/" + version

const headerUserAgent = "User-Agent"

type (
	HTTPTransport struct {
		options        HTTPTransportOptions
		serviceBaseURL *url.URL
	}

	HTTPTransportOptions struct {
		// Base URL for all requests. Required.
		BaseURL string
		// A function for making HTTP requests.
		// Defaults to [http.DefaultClient.Do].
		HTTPCaller func(*http.Request) (*http.Response, error)
		// A [Serializer] to customize client serialization behavior.
		// By default the client handles JSONables, byte slices, and nil.
		Serializer Serializer
		// A [FailureConverter] to convert a [Failure] instance to and from an [error]. Defaults to
		// [DefaultFailureConverter].
		FailureConverter FailureConverter
		// UseOperationID instructs the client to use an older format of the protocol where operation ID is sent
		// as part of the URL path.
		// This flag will be removed in a future release.
		//
		// NOTE: Experimental
		UseOperationID bool
	}
)

func NewHTTPTransport(options HTTPTransportOptions) (*HTTPTransport, error) {
	if options.HTTPCaller == nil {
		options.HTTPCaller = http.DefaultClient.Do
	}
	if options.BaseURL == "" {
		return nil, errors.New("empty BaseURL")
	}
	var baseURL *url.URL
	var err error
	baseURL, err = url.Parse(options.BaseURL)
	if err != nil {
		return nil, err
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme: %s", baseURL.Scheme)
	}
	if options.Serializer == nil {
		options.Serializer = defaultSerializer
	}
	if options.FailureConverter == nil {
		options.FailureConverter = defaultFailureConverter
	}

	return &HTTPTransport{
		options:        options,
		serviceBaseURL: baseURL,
	}, nil
}

func (t *HTTPTransport) StartOperation(
	ctx context.Context,
	service string,
	operation string,
	input any,
	options StartOperationOptions,
) (*StartOperationResponse[*LazyValue], error) {
	var reader *Reader
	if r, ok := input.(*Reader); ok {
		// Close the input reader in case we error before sending the HTTP request (which may double close but
		// that's fine since we ignore the error).
		defer r.Close()
		reader = r
	} else {
		content, ok := input.(*Content)
		if !ok {
			var err error
			content, err = t.options.Serializer.Serialize(input)
			if err != nil {
				return nil, err
			}
		}
		header := maps.Clone(content.Header)
		if header == nil {
			header = make(Header, 1)
		}
		header["length"] = strconv.Itoa(len(content.Data))

		reader = &Reader{
			io.NopCloser(bytes.NewReader(content.Data)),
			header,
		}
	}

	u := t.serviceBaseURL.JoinPath(url.PathEscape(service), url.PathEscape(operation))

	if options.CallbackURL != "" {
		q := u.Query()
		q.Set(queryCallbackURL, options.CallbackURL)
		u.RawQuery = q.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, "POST", u.String(), reader)
	if err != nil {
		return nil, err
	}

	if options.RequestID == "" {
		options.RequestID = uuid.NewString()
	}
	request.Header.Set(headerRequestID, options.RequestID)
	request.Header.Set(headerUserAgent, userAgent)
	addContentHeaderToHTTPHeader(reader.Header, request.Header)
	addCallbackHeaderToHTTPHeader(options.CallbackHeader, request.Header)
	if err := addLinksToHTTPHeader(options.Links, request.Header); err != nil {
		return nil, fmt.Errorf("failed to serialize links into header: %w", err)
	}
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	addNexusHeaderToHTTPHeader(options.Header, request.Header)

	response, err := t.options.HTTPCaller(request)
	if err != nil {
		return nil, err
	}

	links, err := getLinksFromHeader(response.Header)
	if err != nil {
		// Have to read body here to check if it is a Failure.
		body, err := readAndReplaceBody(response)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(
			"%w: %w",
			newUnexpectedResponseError(
				fmt.Sprintf("invalid links header: %q", response.Header.Values(headerLink)),
				response,
				body,
			),
			err,
		)
	}

	// Do not close response body here to allow successful result to read it.
	if response.StatusCode == http.StatusOK {
		return &StartOperationResponse[*LazyValue]{
			Complete: &OperationResult[*LazyValue]{
				result: &LazyValue{
					serializer: t.options.Serializer,
					Reader: &Reader{
						response.Body,
						prefixStrippedHTTPHeaderToNexusHeader(response.Header, "content-"),
					},
				},
			},
			Links: links,
		}, nil
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusCreated:
		info, err := operationInfoFromResponse(response, body)
		if err != nil {
			return nil, err
		}
		if info.State != OperationStateRunning {
			return nil, newUnexpectedResponseError(fmt.Sprintf("invalid operation state in response info: %q", info.State), response, body)
		}
		if info.Token == "" && info.ID != "" {
			info.Token = info.ID
		}
		if info.Token == "" {
			return nil, newUnexpectedResponseError("empty operation token in response", response, body)
		}
		handle := &OperationHandle[*LazyValue]{
			Service:   service,
			Operation: operation,
			Token:     info.Token,
			transport: t,
		}
		return &StartOperationResponse[*LazyValue]{
			Pending: handle,
			Links:   links,
		}, nil
	case statusOperationFailed:
		state, err := getUnsuccessfulStateFromHeader(response, body)
		if err != nil {
			return nil, err
		}

		failure, err := t.failureFromResponse(response, body)
		if err != nil {
			return nil, err
		}

		failureErr := t.options.FailureConverter.FailureToError(failure)
		return nil, &OperationError{
			State: state,
			Cause: failureErr,
		}
	default:
		return nil, t.bestEffortHandlerErrorFromResponse(response, body)
	}
}

func (t *HTTPTransport) GetOperationInfo(
	ctx context.Context,
	service string,
	operation string,
	token string,
	options GetOperationInfoOptions,
) (*GetOperationInfoResponse, error) {
	u := t.serviceBaseURL.JoinPath(url.PathEscape(service), url.PathEscape(operation))
	request, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set(HeaderOperationToken, token)
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	addNexusHeaderToHTTPHeader(options.Header, request.Header)
	request.Header.Set(headerUserAgent, userAgent)

	response, err := t.options.HTTPCaller(request)
	if err != nil {
		return nil, err
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, t.bestEffortHandlerErrorFromResponse(response, body)
	}

	info, err := operationInfoFromResponse(response, body)
	if err != nil {
		return nil, err
	}

	return &GetOperationInfoResponse{
		Info: info,
	}, nil
}

func (t *HTTPTransport) GetOperationResult(
	ctx context.Context,
	service string,
	operation string,
	token string,
	options GetOperationResultOptions,
) (*GetOperationResultResponse[*LazyValue], error) {
	u := t.serviceBaseURL.JoinPath(url.PathEscape(service), url.PathEscape(operation), "result")
	request, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set(HeaderOperationToken, token)
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	request.Header.Set(headerUserAgent, userAgent)
	addNexusHeaderToHTTPHeader(options.Header, request.Header)

	startTime := time.Now()
	wait := options.Wait
	for {
		if wait > 0 {
			if deadline, set := ctx.Deadline(); set {
				// Ensure we don't wait longer than the deadline but give some buffer to prevent racing between wait and
				// context deadline.
				wait = min(wait, time.Until(deadline)+getResultContextPadding)
			}

			q := request.URL.Query()
			q.Set(queryWait, formatDuration(wait))
			request.URL.RawQuery = q.Encode()
		} else {
			// We may reuse the request object multiple times and will need to reset the query when wait becomes 0 or
			// negative.
			request.URL.RawQuery = ""
		}

		response, err := t.sendGetOperationResultRequest(request)
		if err != nil {
			if wait > 0 && errors.Is(err, errOperationWaitTimeout) {
				// TODO: Backoff a bit in case the server is continually returning timeouts due to some LB configuration
				// issue to avoid blowing it up with repeated calls.
				wait = options.Wait - time.Since(startTime)
				continue
			}
			return nil, err
		}
		links, err := getLinksFromHeader(response.Header)
		if err != nil {
			return nil, err
		}
		return &GetOperationResultResponse[*LazyValue]{
			result: &OperationResult[*LazyValue]{
				result: &LazyValue{
					serializer: t.options.Serializer,
					Reader: &Reader{
						response.Body,
						prefixStrippedHTTPHeaderToNexusHeader(response.Header, "content-"),
					},
				},
			},
			Links: links,
		}, nil
	}
}

func (t *HTTPTransport) sendGetOperationResultRequest(request *http.Request) (*http.Response, error) {
	response, err := t.options.HTTPCaller(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusOK {
		return response, nil
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusRequestTimeout:
		return nil, errOperationWaitTimeout
	case statusOperationRunning:
		return nil, ErrOperationStillRunning
	case statusOperationFailed:
		state, err := getUnsuccessfulStateFromHeader(response, body)
		if err != nil {
			return nil, err
		}
		failure, err := t.failureFromResponse(response, body)
		if err != nil {
			return nil, err
		}
		failureErr := t.options.FailureConverter.FailureToError(failure)
		return nil, &OperationError{
			State: state,
			Cause: failureErr,
		}
	default:
		return nil, t.bestEffortHandlerErrorFromResponse(response, body)
	}
}

func (t *HTTPTransport) CancelOperation(
	ctx context.Context,
	service string,
	operation string,
	token string,
	options CancelOperationOptions,
) (*CancelOperationResponse, error) {
	u := t.serviceBaseURL.JoinPath(url.PathEscape(service), url.PathEscape(operation), "cancel")
	request, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set(HeaderOperationToken, token)
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	request.Header.Set(headerUserAgent, userAgent)
	addNexusHeaderToHTTPHeader(options.Header, request.Header)

	response, err := t.options.HTTPCaller(request)
	if err != nil {
		return nil, err
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusAccepted {
		return nil, t.bestEffortHandlerErrorFromResponse(response, body)
	}
	return &CancelOperationResponse{}, nil
}

func (t *HTTPTransport) failureFromResponse(response *http.Response, body []byte) (Failure, error) {
	if !isMediaTypeJSON(response.Header.Get("Content-Type")) {
		return Failure{}, newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get("Content-Type")), response, body)
	}
	var failure Failure
	err := json.Unmarshal(body, &failure)
	return failure, err
}

func (t *HTTPTransport) failureFromResponseOrDefault(response *http.Response, body []byte, defaultMessage string) Failure {
	failure, err := t.failureFromResponse(response, body)
	if err != nil {
		failure.Message = defaultMessage
	}
	return failure
}

func (t *HTTPTransport) failureErrorFromResponseOrDefault(response *http.Response, body []byte, defaultMessage string) error {
	failure := t.failureFromResponseOrDefault(response, body, defaultMessage)
	failureErr := t.options.FailureConverter.FailureToError(failure)
	return failureErr
}

func (t *HTTPTransport) bestEffortHandlerErrorFromResponse(response *http.Response, body []byte) error {
	switch response.StatusCode {
	case http.StatusBadRequest:
		return &HandlerError{
			Type:          HandlerErrorTypeBadRequest,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "bad request"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case http.StatusUnauthorized:
		return &HandlerError{
			Type:          HandlerErrorTypeUnauthenticated,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "unauthenticated"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case http.StatusForbidden:
		return &HandlerError{
			Type:          HandlerErrorTypeUnauthorized,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "unauthorized"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case http.StatusNotFound:
		return &HandlerError{
			Type:          HandlerErrorTypeNotFound,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "not found"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case http.StatusTooManyRequests:
		return &HandlerError{
			Type:          HandlerErrorTypeResourceExhausted,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "resource exhausted"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case http.StatusInternalServerError:
		return &HandlerError{
			Type:          HandlerErrorTypeInternal,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "internal error"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case http.StatusNotImplemented:
		return &HandlerError{
			Type:          HandlerErrorTypeNotImplemented,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "not implemented"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case http.StatusServiceUnavailable:
		return &HandlerError{
			Type:          HandlerErrorTypeUnavailable,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "unavailable"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	case StatusUpstreamTimeout:
		return &HandlerError{
			Type:          HandlerErrorTypeUpstreamTimeout,
			Cause:         t.failureErrorFromResponseOrDefault(response, body, "upstream timeout"),
			RetryBehavior: retryBehaviorFromHeader(response.Header),
		}
	default:
		return newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response, body)
	}
}

// readAndReplaceBody reads the response body in its entirety and closes it, and then replaces the original response
// body with an in-memory buffer.
// The body is replaced even when there was an error reading the entire body.
func readAndReplaceBody(response *http.Response) ([]byte, error) {
	responseBody := response.Body
	body, err := io.ReadAll(responseBody)
	responseBody.Close()
	response.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
}

func operationInfoFromResponse(response *http.Response, body []byte) (*OperationInfo, error) {
	if !isMediaTypeJSON(response.Header.Get("Content-Type")) {
		return nil, newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get("Content-Type")), response, body)
	}
	var info OperationInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func retryBehaviorFromHeader(header http.Header) HandlerErrorRetryBehavior {
	switch strings.ToLower(header.Get(headerRetryable)) {
	case "true":
		return HandlerErrorRetryBehaviorRetryable
	case "false":
		return HandlerErrorRetryBehaviorNonRetryable
	default:
		return HandlerErrorRetryBehaviorUnspecified
	}
}

func getUnsuccessfulStateFromHeader(response *http.Response, body []byte) (OperationState, error) {
	state := OperationState(response.Header.Get(headerOperationState))
	switch state {
	case OperationStateCanceled:
		return state, nil
	case OperationStateFailed:
		return state, nil
	default:
		return state, newUnexpectedResponseError(fmt.Sprintf("invalid operation state header: %q", state), response, body)
	}
}
