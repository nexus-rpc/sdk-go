package nexusclient

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
		{
			message: "with payload",
			details: Payload{Metadata: map[string]string{}, Data: []byte{0x00, 0x01}},
			serialized: `{
	"message": "with payload",
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
		t.Run(tc.message, func(t *testing.T) {
			source, err := MarshalFailureIndent(tc.message, tc.details, "", "\t")
			require.NoError(t, err)
			require.Equal(t, tc.serialized, string(source))
			var failure Failure
			err = json.Unmarshal(source, &failure)
			require.NoError(t, err)
			detailsPointer := reflect.New(reflect.TypeOf(tc.details)).Interface()
			err = failure.UnmarshalDetails(detailsPointer)
			details := reflect.ValueOf(detailsPointer).Elem().Interface()
			require.NoError(t, err)
			require.Equal(t, tc.details, details)
		})
	}
}

func TestPayload_JSONMarshalling(t *testing.T) {
	// TODO: test transfer encoding cases
}
