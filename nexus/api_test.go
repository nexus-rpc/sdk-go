package nexus

import (
	"encoding/json"
	"net/http"
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

func TestAddLinksToHeader(t *testing.T) {
	type testcase struct {
		name   string
		input  []Link
		output http.Header
		errMsg string
	}

	cases := []testcase{
		{
			name: "single link",
			input: []Link{{
				URL:  "https://example.com/path?param=value",
				Type: "url",
			}},
			output: http.Header{
				headerLink: []string{
					`<https://example.com/path?param=value>; type="url"`,
				},
			},
		},
		{
			name: "multiple links",
			input: []Link{
				{
					URL:  "https://example.com/path?param=value",
					Type: "url",
				},
				{
					URL:  "https://foo.com/path?bar=value",
					Type: "url",
				},
			},
			output: http.Header{
				headerLink: []string{
					`<https://example.com/path?param=value>; type="url"`,
					`<https://foo.com/path?bar=value>; type="url"`,
				},
			},
		},
		{
			name: "invalid link",
			input: []Link{{
				URL:  "https://example.com/path%",
				Type: "url",
			}},
			errMsg: "failed to encode link",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := http.Header{}
			err := addLinksToHTTPHeader(tc.input, output)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.output, output)
			}
		})
	}
}

func TestGetLinksFromHeader(t *testing.T) {
	type testcase struct {
		name   string
		input  http.Header
		output []Link
		errMsg string
	}

	cases := []testcase{
		{
			name: "single link",
			input: http.Header{
				headerLink: []string{
					`<https://example.com/path?param=value>; type="url"`,
				},
			},
			output: []Link{{
				URL:  "https://example.com/path?param=value",
				Type: "url",
			}},
		},
		{
			name: "multiple links",
			input: http.Header{
				headerLink: []string{
					`<https://example.com/path?param=value>; type="url"`,
					`<https://foo.com/path?bar=value>; type="url"`,
				},
			},
			output: []Link{
				{
					URL:  "https://example.com/path?param=value",
					Type: "url",
				},
				{
					URL:  "https://foo.com/path?bar=value",
					Type: "url",
				},
			},
		},
		{
			name: "multiple links single header",
			input: http.Header{
				headerLink: []string{
					`<https://example.com/path?param=value>; type="url", <https://foo.com/path?bar=value>; type="url"`,
				},
			},
			output: []Link{
				{
					URL:  "https://example.com/path?param=value",
					Type: "url",
				},
				{
					URL:  "https://foo.com/path?bar=value",
					Type: "url",
				},
			},
		},
		{
			name: "invalid header",
			input: http.Header{
				headerLink: []string{
					`<https://example.com/path?param=value> Type="url"`,
				},
			},
			errMsg: "failed to parse link header value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := getLinksFromHeader(tc.input)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.output, output)
			}
		})
	}
}

func TestEncodeLink(t *testing.T) {
	type testcase struct {
		name   string
		input  Link
		output string
		errMsg string
	}

	cases := []testcase{
		{
			name: "valid",
			input: Link{
				URL:  "https://example.com/path/to/something?param1=value1&param2=value2",
				Type: "text/plain",
			},
			output: `<https://example.com/path/to/something?param1=value1&param2=value2>; type="text/plain"`,
		},
		{
			name: "valid with encoded percent and semi-colon",
			input: Link{
				URL:  "https://example.com/path/to/something%25%3B?param1=value1&param2=value2",
				Type: "text/plain",
			},
			output: `<https://example.com/path/to/something%25%3B?param1=value1&param2=value2>; type="text/plain"`,
		},
		{
			name: "valid custom URL",
			input: Link{
				URL:  "nexus:///path/to/something?param1=value",
				Type: "nexus.data_type",
			},
			output: `<nexus:///path/to/something?param1=value>; type="nexus.data_type"`,
		},
		{
			name: "invalid url empty",
			input: Link{
				URL:  "",
				Type: "text/plain",
			},
			errMsg: "failed to encode link",
		},
		{
			name: "invalid url",
			input: Link{
				URL:  "https://example.com/path/to/something%?param1=value1&param2=value2",
				Type: "text/plain",
			},
			errMsg: "failed to encode link",
		},
		{
			name: "invalid type empty",
			input: Link{
				URL:  "https://example.com/path/to/something?param1=value1&param2=value2",
				Type: "",
			},
			errMsg: "failed to encode link",
		},
		{
			name: "invalid path not percent-encoded ;",
			input: Link{
				URL:  "https://example.com/path/to/something%3B;?param1=value1&param2=value2",
				Type: "text/plain",
			},
			errMsg: "failed to encode link",
		},
		{
			name: "invalid path not percent-encoded ,",
			input: Link{
				URL:  "https://example.com/path/to/something,?param1=value1&param2=value2",
				Type: "text/plain",
			},
			errMsg: "failed to encode link",
		},
		{
			name: "invalid query not percent-encoded",
			input: Link{
				URL:  "https://example.com/path/to/something?param1=value1&param2=value2;",
				Type: "text/plain",
			},
			errMsg: "failed to encode link",
		},
		{
			name: "invalid type invalid chars",
			input: Link{
				URL:  "https://example.com/path/to/something?param1=value1&param2=value2",
				Type: "text/plain;",
			},
			errMsg: "failed to encode link",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := encodeLink(tc.input)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.output, output)
			}
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
			input: `<https://example.com/path/to/something?param1=value1&param2=value2>; type="text/plain"`,
			output: Link{
				URL:  "https://example.com/path/to/something?param1=value1&param2=value2",
				Type: "text/plain",
			},
		},
		{
			name:  "valid multiple params",
			input: `<https://example.com/path/to/something?param1=value1&param2=value2>; type="text/plain"; Param="value"`,
			output: Link{
				URL:  "https://example.com/path/to/something?param1=value1&param2=value2",
				Type: "text/plain",
			},
		},
		{
			name:  "valid param not quoted",
			input: `<https://example.com/path/to/something?param1=value1&param2=value2>; type=text/plain`,
			output: Link{
				URL:  "https://example.com/path/to/something?param1=value1&param2=value2",
				Type: "text/plain",
			},
		},
		{
			name:  "valid custom URL",
			input: `<nexus:///path/to/something?param=value>; type="nexus.data_type"`,
			output: Link{
				URL:  "nexus:///path/to/something?param=value",
				Type: "nexus.data_type",
			},
		},
		{
			name:   "invalid url",
			input:  `<https://example.com/path/to/something%?param1=value1&param2=value2>`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid trailing semi-colon",
			input:  `<https://example.com/path/to/something?param1=value1&param2=value2>; type="text/plain";`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid empty param part",
			input:  `<https://example.com/path/to/something?param1=value1&param2=value2>; ; type="text/plain`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid no type param trailing semi-colon",
			input:  `<https://example.com/path/to/something?param1=value1&param2=value2>;`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid url not enclosed",
			input:  `https://example.com/path/to/something?param1=value1&param2=value2; type="text/plain"`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid url missing closing bracket",
			input:  `<https://example.com/path/to/something?param1=value1&param2=value2; type="text/plain"`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid url missing opening bracket",
			input:  `https://example.com/path/to/something?param1=value1&param2=value2>; type="text/plain"`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid param missing quote",
			input:  `https://example.com/path/to/something?param1=value1&param2=value2>; type="text/plain`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid multiple params missing semi-colon",
			input:  `https://example.com/path/to/something?param1=value1&param2=value2>; type="text/plain" Param=value`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid missing semi-colon after url",
			input:  `https://example.com/path/to/something?param1=value1&param2=value2> type="text/plain"`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid param missing value",
			input:  `https://example.com/path/to/something?param1=value1&param2=value2>; type`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid param missing value with equal sign",
			input:  `<https://example.com/path/to/something?param1=value1&param2=value2>; type=`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid missing type key",
			input:  `<https://example.com/path/to/something?param1=value1&param2=value2>`,
			errMsg: "failed to parse link header value",
		},
		{
			name:   "invalid url empty",
			input:  `<>; type="text/plain"`,
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
