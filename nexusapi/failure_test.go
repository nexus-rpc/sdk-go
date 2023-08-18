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
		metadata   map[string]string
		serialized string
	}
	cases := []testcase{
		{
			message:  "simple",
			details:  "details",
			metadata: map[string]string{},
			serialized: `{
	"message": "simple",
	"metadata": {},
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
			source, err := json.MarshalIndent(Failure{tc.message, tc.metadata, serializedDetails}, "", "\t")
			require.NoError(t, err)
			require.Equal(t, tc.serialized, string(source))

			var failure Failure
			err = json.Unmarshal(source, &failure)
			require.NoError(t, err)

			require.Equal(t, tc.message, failure.Message)
			require.Equal(t, tc.metadata, failure.Metadata)

			detailsPointer := reflect.New(reflect.TypeOf(tc.details)).Interface()
			err = json.Unmarshal(failure.Details, detailsPointer)
			details := reflect.ValueOf(detailsPointer).Elem().Interface()
			require.NoError(t, err)
			require.Equal(t, tc.details, details)
		})
	}
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
