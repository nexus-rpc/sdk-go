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

	Failure struct {
		Message string          `json:"message"`
		Details json.RawMessage `json:"details"`
	}
)

const (
	contentTransferEncoding      = "Content-Transfer-Encoding"
	contentTransferEncodingLower = "content-transfer-encoding"
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

func MarshalFailure(message string, details any) ([]byte, error) {
	data, err := json.Marshal(details)
	if err != nil {
		return nil, err
	}
	failure := &Failure{
		Message: message,
		Details: data,
	}
	return json.Marshal(failure)
}

func MarshalFailureIndent(message string, details any, prefix string, indent string) ([]byte, error) {
	data, err := json.MarshalIndent(details, prefix, indent)
	if err != nil {
		return nil, err
	}
	failure := &Failure{
		Message: message,
		Details: data,
	}
	return json.MarshalIndent(failure, prefix, indent)
}

func (f *Failure) UnmarshalDetails(v any) error {
	return json.Unmarshal(f.Details, v)
}
