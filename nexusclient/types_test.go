package nexusclient

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailure_JSONMarshalling(t *testing.T) {
	type testcase struct {
		name       string
		message    any
		details    any
		serialized string
	}
	cases := []testcase{
		{
			name:    "simple",
			message: "simple",
			details: "details",
			serialized: `{
	"message": "simple",
	"details": "details"
}`,
		},
		{
			name:    "complex",
			message: "complex",
			details: struct {
				M map[string]string
				I int64
			}{
				M: map[string]string{"a": "b"},
				I: 654,
			},
			serialized: `{
	"message": "complex",
	"details": {
		"M": {
			"a": "b"
		},
		"I": 654
	}
}`,
		},
		{
			name:    "with payload",
			message: Payload{Metadata: map[string]string{}, Data: []byte("message")},
			details: Payload{Metadata: map[string]string{}, Data: []byte{0x00, 0x01}},
			serialized: `{
	"message": {
		"metadata": {
			"Content-Transfer-Encoding": "base64"
		},
		"data": "bWVzc2FnZQ=="
	},
	"details": {
		"metadata": {
			"Content-Transfer-Encoding": "base64"
		},
		"data": "AAE="
	}
}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			source, err := json.MarshalIndent(UntypedFailure{tc.message, tc.details}, "", "\t")
			require.NoError(t, err)
			require.Equal(t, tc.serialized, string(source))

			var failure Failure
			err = json.Unmarshal(source, &failure)
			require.NoError(t, err)

			messagePointer := reflect.New(reflect.TypeOf(tc.message)).Interface()
			err = json.Unmarshal(failure.Message, messagePointer)
			message := reflect.ValueOf(messagePointer).Elem().Interface()
			require.NoError(t, err)
			require.Equal(t, tc.message, message)

			detailsPointer := reflect.New(reflect.TypeOf(tc.details)).Interface()
			err = json.Unmarshal(failure.Details, detailsPointer)
			details := reflect.ValueOf(detailsPointer).Elem().Interface()
			require.NoError(t, err)
			require.Equal(t, tc.details, details)
		})
	}
}

func TestPayload_JSONMarshalling(t *testing.T) {
	// TODO: test transfer encoding cases
}

func TestOperationInfo_JSONMarshalling(t *testing.T) {
	data, err := json.Marshal(OperationInfo{ID: "abc", State: OperationStateCanceled})
	require.NoError(t, err)
	require.Equal(t, `{"id":"abc","state":"canceled"}`, string(data))

	var info OperationInfo

	err = json.Unmarshal([]byte(`{"id":"def","state":"succeeded"}`), &info)
	require.NoError(t, err)
	require.Equal(t, OperationInfo{ID: "def", State: OperationStateSucceeded}, info)

	err = json.Unmarshal([]byte(`{"id":"def","state":"unknown"}`), &info)
	require.NoError(t, err)
	require.Equal(t, OperationInfo{ID: "def", State: OperationState("unknown")}, info)
}
