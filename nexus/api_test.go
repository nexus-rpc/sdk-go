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
		details    any
		metadata   map[string]string
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

func TestEncodeLink(t *testing.T) {
	type testcase struct {
		name     string
		input    Link
		expected string
	}

	cases := []testcase{
		{
			name: "valid",
			input: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "temporal.api.common.v1.Link.WorkflowEvent",
			},
			expected: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type="temporal.api.common.v1.Link.WorkflowEvent"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := encodeLink(tc.input)
			require.Equal(t, tc.expected, output)
		})
	}
}

func TestDecodeLink(t *testing.T) {
	type testcase struct {
		name   string
		input  string
		output Link
		errMsg string
	}

	cases := []testcase{
		{
			name:  "valid",
			input: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type="temporal.api.common.v1.Link.WorkflowEvent"`,
			output: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "temporal.api.common.v1.Link.WorkflowEvent",
			},
		},
		{
			name:  "valid multiple params",
			input: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type="temporal.api.common.v1.Link.WorkflowEvent"; Param="value"`,
			output: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "temporal.api.common.v1.Link.WorkflowEvent",
			},
		},
		{
			name:  "valid param not quoted",
			input: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type=temporal.api.common.v1.Link.WorkflowEvent`,
			output: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "temporal.api.common.v1.Link.WorkflowEvent",
			},
		},
		{
			name:  "valid param missing value with equal sign",
			input: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type=;`,
			output: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "",
			},
		},
		{
			name:  "valid no type param",
			input: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>`,
			output: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "",
			},
		},
		{
			name:  "valid trailing semi-colon",
			input: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type="temporal.api.common.v1.Link.WorkflowEvent";`,
			output: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "temporal.api.common.v1.Link.WorkflowEvent",
			},
		},
		{
			name:  "valid no type param trailing semi-colon",
			input: `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>;`,
			output: Link{
				URL:  "temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2",
				Type: "",
			},
		},
		{
			name:   "invalid url not enclosed",
			input:  `temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid url missing closing bracket",
			input:  `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid url missing opening bracket",
			input:  `temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid param",
			input:  `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type="temporal.api.common.v1.Link.WorkflowEvent`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid multiple params missing semi-colon",
			input:  `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type="temporal.api.common.v1.Link.WorkflowEvent" Param=value`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid missing semi-colon after url",
			input:  `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2> Type="temporal.api.common.v1.Link.WorkflowEvent";`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid param missing value",
			input:  `<temporal:///namespaces/test-ns/workflows/test-wf-id/test-run-id/history?event_id=1&event_type=2>; Type;`,
			errMsg: "failed to parse link header value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := decodeLink(tc.input)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.output, output)
			}
		})
	}
}
