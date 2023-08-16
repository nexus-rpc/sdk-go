package nexusapi

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailure_JSONMarshalling(t *testing.T) {
	type testcase struct {
		message    string
		details    any
		serialized string
	}
	cases := []testcase{
		{
			message: "simple",
			details: "details",
			serialized: `{
	"message": "simple",
	"details": "details"
}`,
		},
		{
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
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.message, func(t *testing.T) {
			source, err := json.MarshalIndent(Failure{tc.message, tc.details}, "", "\t")
			require.NoError(t, err)
			require.Equal(t, tc.serialized, string(source))

			var failure RawFailure
			err = json.Unmarshal(source, &failure)
			require.NoError(t, err)

			require.Equal(t, tc.message, failure.Message)

			detailsPointer := reflect.New(reflect.TypeOf(tc.details)).Interface()
			err = json.Unmarshal(failure.Details, detailsPointer)
			details := reflect.ValueOf(detailsPointer).Elem().Interface()
			require.NoError(t, err)
			require.Equal(t, tc.details, details)
		})
	}
}

func TestPayloadFailure_JSONMarshalling(t *testing.T) {
	failure := PayloadFailure{
		Message: Payload{Metadata: map[string]string{"a": "b"}, Data: []byte("message")},
		Details: Payload{Metadata: map[string]string{}, Data: []byte{0x00, 0x01}},
	}
	serialized := `{
	"message": {
		"metadata": {
			"a": "b"
		},
		"data": "bWVzc2FnZQ=="
	},
	"details": {
		"metadata": {},
		"data": "AAE="
	}
}`
	source, err := json.MarshalIndent(failure, "", "\t")
	require.NoError(t, err)
	require.Equal(t, serialized, string(source))

	var decoded PayloadFailure
	err = json.Unmarshal(source, &decoded)
	require.NoError(t, err)

	require.Equal(t, failure.Message, decoded.Message)
	require.Equal(t, failure.Details, decoded.Details)
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
