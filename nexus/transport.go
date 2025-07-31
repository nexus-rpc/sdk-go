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

// Transport is the low-level abstraction used by [ServiceClient] and [OperationHandle] to make network calls.
//
// Implementations must embed [UnimplementedTransport] for future compatibility.
//
// NOTE: Experimental
type Transport interface {

	// StartOperation calls the configured Nexus endpoint to start an operation.
	//
	// This method has the following possible outcomes:
	//
	//  1. The operation completes successfully. The result of this call will be set as a [LazyValue] in
	//     TransportStartOperationResponse.Complete and must be consumed to free up the underlying connection.
	//
	//	2. The operation completes unsuccessfully. The response will contain an [OperationError] in
	//	   TransportStartOperationResponse.Complete
	//
	//  3. The operation was started and the handler has indicated that it will complete asynchronously. An
	//     [OperationHandle] will be returned as TransportStartOperationResponse.Pending, which can be used to perform actions
	//     such as getting its result.
	//
	//  4. There was an error making the call. The returned response will be nil and the error will be non-nil.
	//     Most often it will be a [HandlerError].
	//
	// NOTE: Experimental
	StartOperation(ctx context.Context, input any, options TransportStartOperationOptions) (*TransportStartOperationResponse[*LazyValue], error)

	// GetOperationInfo returns information about a specific operation.
	//
	// NOTE: Experimental
	GetOperationInfo(ctx context.Context, options TransportGetOperationInfoOptions) (*TransportGetOperationInfoResponse, error)

	// GetOperationResult gets the result of an operation, issuing a network request to the service handler.
	//
	// By default, GetResult returns (nil, [ErrOperationStillRunning]) immediately after issuing a call if the
	// operation has not yet completed.
	//
	// Callers may set GetOperationResultOptions.Wait to a value greater than 0 to alter this behavior, causing
	// the transport to long poll for the result issuing one or more requests until the provided wait period
	// exceeds, in which case (nil, [ErrOperationStillRunning]) is returned.
	//
	// The wait time is capped to the deadline of the provided context. Make sure to handle both context deadline
	// errors and [ErrOperationStillRunning].
	//
	// Note that the wait period is enforced by the server and may not be respected if the server is misbehaving.
	// Set the context deadline to the max allowed wait period to ensure this call returns in a timely fashion.
	//
	// ⚠️ If a [LazyValue] is returned (as indicated by T), it must be consumed to free up the underlying connection.
	//
	// NOTE: Experimental
	GetOperationResult(ctx context.Context, options TransportGetOperationResultOptions) (*TransportGetOperationResultResponse[*LazyValue], error)

	// CancelOperation requests to cancel an asynchronous operation.
	//
	// Cancelation is asynchronous and may be not be respected by the operation's implementation.
	//
	// NOTE: Experimental
	CancelOperation(ctx context.Context, options TransportCancelOperationOptions) (*TransportCancelOperationResponse, error)

	// Close this Transport and release any underlying resources.
	//
	// NOTE: Experimental
	Close() error

	// UnimplementedTransport must be embedded into any [Transport] implementation for future compatibility.
	// It implements all methods on the [Transport] interface, returning unimplemented errors if they are not implemented by
	// the embedding type.
	mustEmbedUnimplementedTransport()
}

// TransportError indicates a client encountered something unexpected in the server's response.
type TransportError struct {
	// Error message.
	Message string
	// Optional failure that may have been embedded in the response.
	Failure *Failure
	// Additional transport specific details.
	// For HTTP, this would include the HTTP response. The response body will have already been read into memory and
	// does not need to be closed.
	Details any
}

// Error implements the error interface.
func (e *TransportError) Error() string {
	return e.Message
}

func newTransportError(message string, response *http.Response, body []byte) error {
	var failure *Failure
	if isMediaTypeJSON(response.Header.Get("Content-Type")) {
		if err := json.Unmarshal(body, &failure); err == nil && failure.Message != "" {
			message += ": " + failure.Message
		}
	}

	return &TransportError{
		Message: message,
		Details: response,
		Failure: failure,
	}
}

// TransportStartOperationResponse is the response to Transport.StartOperation calls. One and only one of Complete or
// Pending will be populated.
//
// NOTE: Experimental
type TransportStartOperationResponse[T any] struct {
	// Set when start completes synchronously.
	//
	// If T is a [LazyValue], ensure that your consume it or read the underlying content in its entirety and close
	// it to free up the underlying connection.
	Complete *OperationResult[T]
	// Set when the handler indicates that it started an asynchronous operation.
	// The attached handle can be used to perform actions such as cancel the operation or get its result.
	Pending *OperationHandle[T]
	// Links contain information about the operations done by the handler.
	Links []Link
}

// TransportGetOperationInfoResponse is the response to Transport.GetOperationInfo calls.
//
// NOTE: Experimental
type TransportGetOperationInfoResponse struct {
	Info *OperationInfo
}

// TransportGetOperationResultResponse is the response to Transport.GetOperationResult calls.
// Use TransportGetOperationResultResponse.GetResult to retrieve the final value or error returned by the operation.
//
// NOTE: Experimental
type TransportGetOperationResultResponse[T any] struct {
	result *OperationResult[T]
	Links  []Link
}

// GetResult returns the final result or error returned by the operation.
//
// NOTE: Experimental
func (gr *TransportGetOperationResultResponse[T]) GetResult() (T, error) {
	return gr.result.Get()
}

// TransportCancelOperationResponse is the response to Transport.CancelOperation calls.
//
// NOTE: Experimental
type TransportCancelOperationResponse struct{}

// OperationResult contains the final value or error returned by an operation handler. One and only one of
// result or err will be populated. Use OperationResult.Get to retrieve the result.
//
// In most cases err will be an [OperationError]. Other failures, such as [HandlerError], will be returned by
// the Transport method called to indicate a failure to communicate with the operation handler.
//
// NOTE: Experimental
type OperationResult[T any] struct {
	result T
	err    error
}

// Get returns the final result or error returned by an operation. Only one of result or err should be non-zero/non-nil.
//
// NOTE: Experimental
func (r *OperationResult[T]) Get() (T, error) {
	return r.result, r.err
}

// UnimplementedTransport must be embedded into any [Transport] implementation for future compatibility.
// It implements all methods on the [Transport] interface, returning unimplemented errors if they are not implemented by
// the embedding type.
type UnimplementedTransport struct{}

func (u UnimplementedTransport) mustEmbedUnimplementedTransport() {}

func (u UnimplementedTransport) StartOperation(_ context.Context, _ any, _ TransportStartOperationOptions) (*TransportStartOperationResponse[*LazyValue], error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

func (u UnimplementedTransport) GetOperationInfo(_ context.Context, _ TransportGetOperationInfoOptions) (*TransportGetOperationInfoResponse, error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

func (u UnimplementedTransport) GetOperationResult(_ context.Context, _ TransportGetOperationResultOptions) (*TransportGetOperationResultResponse[*LazyValue], error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

func (u UnimplementedTransport) CancelOperation(_ context.Context, _ TransportCancelOperationOptions) (*TransportCancelOperationResponse, error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

func (u UnimplementedTransport) Close() error {
	return HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// User-Agent header set on HTTP requests.
const userAgent = "Nexus-go-sdk/" + version
const headerUserAgent = "User-Agent"
const getResultContextPadding = time.Second * 5

var errOperationWaitTimeout = errors.New("operation wait timeout")

// HTTPTransport is a [Transport] implementation backed by HTTP. It makes Nexus service requests as defined in
// the [Nexus HTTP API].
//
// NOTE: Experimental
//
// [Nexus HTTP API]: https://github.com/nexus-rpc/api
type HTTPTransport struct {
	UnimplementedTransport

	options        HTTPTransportOptions
	serviceBaseURL *url.URL
}

// HTTPTransportOptions are the options for constructing a new [HTTPTransport].
//
// NOTE: Experimental
type HTTPTransportOptions struct {
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

// NewHTTPTransport creates a new [Transport] backed by HTTP from the provided [HTTPTransportOptions].
//
// NOTE: Experimental
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

func (t *HTTPTransport) Close() error {
	return nil
}

func (t *HTTPTransport) StartOperation(
	ctx context.Context,
	input any,
	options TransportStartOperationOptions,
) (*TransportStartOperationResponse[*LazyValue], error) {
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

	u := t.serviceBaseURL.JoinPath(url.PathEscape(options.Service), url.PathEscape(options.Operation))

	if options.ClientOptions.CallbackURL != "" {
		q := u.Query()
		q.Set(queryCallbackURL, options.ClientOptions.CallbackURL)
		u.RawQuery = q.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, "POST", u.String(), reader)
	if err != nil {
		return nil, err
	}

	if options.ClientOptions.RequestID == "" {
		options.ClientOptions.RequestID = uuid.NewString()
	}
	request.Header.Set(headerRequestID, options.ClientOptions.RequestID)
	request.Header.Set(headerUserAgent, userAgent)
	addContentHeaderToHTTPHeader(reader.Header, request.Header)
	addCallbackHeaderToHTTPHeader(options.ClientOptions.CallbackHeader, request.Header)
	if err := addLinksToHTTPHeader(options.ClientOptions.Links, request.Header); err != nil {
		return nil, fmt.Errorf("failed to serialize links into header: %w", err)
	}
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	addNexusHeaderToHTTPHeader(options.ClientOptions.Header, request.Header)

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
			newTransportError(
				fmt.Sprintf("invalid links header: %q", response.Header.Values(headerLink)),
				response,
				body,
			),
			err,
		)
	}

	// Do not close response body here to allow successful result to read it.
	if response.StatusCode == http.StatusOK {
		return &TransportStartOperationResponse[*LazyValue]{
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
			return nil, newTransportError(fmt.Sprintf("invalid operation state in response info: %q", info.State), response, body)
		}
		if info.Token == "" && info.ID != "" {
			info.Token = info.ID
		}
		if info.Token == "" {
			return nil, newTransportError("empty operation token in response", response, body)
		}
		handle := &OperationHandle[*LazyValue]{
			Service:   options.Service,
			Operation: options.Operation,
			Token:     info.Token,
			transport: t,
		}
		return &TransportStartOperationResponse[*LazyValue]{
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
	options TransportGetOperationInfoOptions,
) (*TransportGetOperationInfoResponse, error) {
	u := t.serviceBaseURL.JoinPath(url.PathEscape(options.Service), url.PathEscape(options.Operation))
	request, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set(HeaderOperationToken, options.Token)
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	addNexusHeaderToHTTPHeader(options.ClientOptions.Header, request.Header)
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

	return &TransportGetOperationInfoResponse{
		Info: info,
	}, nil
}

func (t *HTTPTransport) GetOperationResult(
	ctx context.Context,
	options TransportGetOperationResultOptions,
) (*TransportGetOperationResultResponse[*LazyValue], error) {
	u := t.serviceBaseURL.JoinPath(url.PathEscape(options.Service), url.PathEscape(options.Operation), "result")
	request, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set(HeaderOperationToken, options.Token)
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	request.Header.Set(headerUserAgent, userAgent)
	addNexusHeaderToHTTPHeader(options.ClientOptions.Header, request.Header)

	startTime := time.Now()
	wait := options.ClientOptions.Wait
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
				wait = options.ClientOptions.Wait - time.Since(startTime)
				continue
			}
			return nil, err
		}
		links, err := getLinksFromHeader(response.Header)
		if err != nil {
			return nil, err
		}
		return &TransportGetOperationResultResponse[*LazyValue]{
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
	options TransportCancelOperationOptions,
) (*TransportCancelOperationResponse, error) {
	u := t.serviceBaseURL.JoinPath(url.PathEscape(options.Service), url.PathEscape(options.Operation), "cancel")
	request, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set(HeaderOperationToken, options.Token)
	addContextTimeoutToHTTPHeader(ctx, request.Header)
	request.Header.Set(headerUserAgent, userAgent)
	addNexusHeaderToHTTPHeader(options.ClientOptions.Header, request.Header)

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
	return &TransportCancelOperationResponse{}, nil
}

func (t *HTTPTransport) failureFromResponse(response *http.Response, body []byte) (Failure, error) {
	if !isMediaTypeJSON(response.Header.Get("Content-Type")) {
		return Failure{}, newTransportError(fmt.Sprintf("invalid response content type: %q", response.Header.Get("Content-Type")), response, body)
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
		return newTransportError(fmt.Sprintf("unexpected response status: %q", response.Status), response, body)
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
		return nil, newTransportError(fmt.Sprintf("invalid response content type: %q", response.Header.Get("Content-Type")), response, body)
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
		return state, newTransportError(fmt.Sprintf("invalid operation state header: %q", state), response, body)
	}
}
