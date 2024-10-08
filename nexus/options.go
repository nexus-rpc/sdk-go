package nexus

import (
	"time"
)

// StartOperationOptions are options for the StartOperation client and server APIs.
type StartOperationOptions struct {
	// Header contains the request header fields either received by the server or to be sent by the client.
	//
	// Header will always be non empty in server methods and can be optionally set in the client API.
	//
	// Header values set here will overwrite any SDK-provided values for the same key.
	//
	// Header keys with the "content-" prefix are reserved for [Serializer] headers and should not be set in the
	// client API; they are not available to server [Handler] and [Operation] implementations.
	Header Header
	// Callbacks are used to deliver completion of async operations.
	// This value may optionally be set by the client and should be called by a handler upon completion if the started operation is async.
	//
	// Implement a [CompletionHandler] and expose it as an HTTP handler to handle async completions.
	CallbackURL string
	// Optional header fields set by a client that are required to be attached to the callback request when an
	// asynchronous operation completes.
	CallbackHeader Header
	// Request ID that may be used by the server handler to dedupe a start request.
	// By default a v4 UUID will be generated by the client.
	RequestID string
	// Links contain arbitrary caller information. Handlers may use these links as
	// metadata on resources associated with and operation.
	Links []Link
}

// GetOperationResultOptions are options for the GetOperationResult client and server APIs.
type GetOperationResultOptions struct {
	// Header contains the request header fields either received by the server or to be sent by the client.
	//
	// Header will always be non empty in server methods and can be optionally set in the client API.
	//
	// Header values set here will overwrite any SDK-provided values for the same key.
	Header Header
	// If non-zero, reflects the duration the caller has indicated that it wants to wait for operation completion,
	// turning the request into a long poll.
	Wait time.Duration
}

// GetOperationInfoOptions are options for the GetOperationInfo client and server APIs.
type GetOperationInfoOptions struct {
	// Header contains the request header fields either received by the server or to be sent by the client.
	//
	// Header will always be non empty in server methods and can be optionally set in the client API.
	//
	// Header values set here will overwrite any SDK-provided values for the same key.
	Header Header
}

// CancelOperationOptions are options for the CancelOperation client and server APIs.
type CancelOperationOptions struct {
	// Header contains the request header fields either received by the server or to be sent by the client.
	//
	// Header will always be non empty in server methods and can be optionally set in the client API.
	//
	// Header values set here will overwrite any SDK-provided values for the same key.
	Header Header
}
