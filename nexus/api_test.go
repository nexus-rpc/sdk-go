package nexus

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailure_JSONMarshalling(t *testing.T) {
	// This test isn't strictly required, it's left to demonstrate how to use Failures.

	type testcase struct {
		message    string
		stackTrace string
		details    any
		metadata   map[string]string
		serialized string
	}
	cases := []testcase{
		{
			message:    "simple",
			stackTrace: "stack",
			details:    "details",
			serialized: `{
	"message": "simple",
	"stackTrace": "stack",
	"details": "details"
}`,
		},
		{
			message:  "complex",
			metadata: map[string]string{"meta": "data"},
			details: struct {
				M map[string]string
				I int64
			}{
				M: map[string]string{"a": "b"},
				I: 654,
			},
			serialized: `{
	"message": "complex",
	"metadata": {
		"meta": "data"
	},
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
			serializedDetails, err := json.MarshalIndent(tc.details, "", "\t")
			require.NoError(t, err)
			source, err := json.MarshalIndent(Failure{tc.message, tc.stackTrace, tc.metadata, serializedDetails, nil}, "", "\t")
			require.NoError(t, err)
			require.Equal(t, tc.serialized, string(source))

			var failure Failure
			err = json.Unmarshal(source, &failure)
			require.NoError(t, err)

			require.Equal(t, tc.message, failure.Message)
			require.Equal(t, tc.stackTrace, failure.StackTrace)
			require.Equal(t, tc.metadata, failure.Metadata)

			detailsPointer := reflect.New(reflect.TypeOf(tc.details)).Interface()
			err = json.Unmarshal(failure.Details, detailsPointer)
			details := reflect.ValueOf(detailsPointer).Elem().Interface()
			require.NoError(t, err)
			require.Equal(t, tc.details, details)
		})
	}
}
