package nexusclient

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

type (
	Payload struct {
		Metadata map[string]string
		Data     []byte
	}

	// jsonPayload is the JSON representation of a Payload.
	// Its data field is base64 encoded.
	jsonPayload struct {
		Metadata map[string]string `json:"metadata"`
		Data     string            `json:"data"`
	}

	// TODO: Find a better name for this
	UntypedFailure struct {
		Message any `json:"message"`
		Details any `json:"details"`
	}

	Failure struct {
		Message json.RawMessage `json:"message"`
		Details json.RawMessage `json:"details"`
	}

	OperationState string

	OperationInfo struct {
		ID    string         `json:"id"`
		State OperationState `json:"state"`
	}
)

const (
	contentTransferEncoding      = "Content-Transfer-Encoding"
	contentTransferEncodingLower = "content-transfer-encoding"
	OperationStateRunning        = OperationState("running")
	OperationStateSucceeded      = OperationState("succeeded")
	OperationStateFailed         = OperationState("failed")
	OperationStateCanceled       = OperationState("canceled")
)

var ErrInvalidPayloadTransferEncoding = errors.New("metadata field Content-Transfer-Encoding not set to base64")

func (p Payload) MarshalJSON() ([]byte, error) {
	jp := jsonPayload{
		Metadata: make(map[string]string, len(p.Metadata)),
		Data:     base64.StdEncoding.EncodeToString(p.Data),
	}
	for k, v := range p.Metadata {
		if strings.ToLower(k) == contentTransferEncodingLower {
			if v != "base64" {
				return nil, ErrInvalidPayloadTransferEncoding
			}
		} else {
			jp.Metadata[k] = v
		}
	}
	jp.Metadata[contentTransferEncoding] = "base64"
	return json.Marshal(jp)
}

func (p *Payload) UnmarshalJSON(data []byte) error {
	var jsonPayload jsonPayload
	if err := json.Unmarshal(data, &jsonPayload); err != nil {
		return err
	}
	for k, v := range jsonPayload.Metadata {
		if strings.ToLower(k) == contentTransferEncodingLower && v == "base64" {
			data, err := base64.StdEncoding.DecodeString(jsonPayload.Data)
			if err != nil {
				return err
			}
			delete(jsonPayload.Metadata, k)
			p.Metadata = jsonPayload.Metadata
			p.Data = data
			return nil
		}
	}
	return ErrInvalidPayloadTransferEncoding
}
