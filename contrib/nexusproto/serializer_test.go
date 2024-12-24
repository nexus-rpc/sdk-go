package nexusproto_test

import (
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/contrib/nexusproto"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestSerializer(t *testing.T) {
	binSerializer := nexusproto.NewSerializer(nexusproto.SerializerOptions{Mode: nexusproto.SerializerModePreferBinary})
	jsonSerializer := nexusproto.NewSerializer(nexusproto.SerializerOptions{Mode: nexusproto.SerializerModePreferJSON})
	d := durationpb.New(time.Minute)

	cases := []struct {
		name         string
		serializer   nexus.Serializer
		deserializer nexus.Serializer
	}{
		{
			name:         "JSON2JSON",
			serializer:   jsonSerializer,
			deserializer: jsonSerializer,
		},
		{
			name:         "JSON2Binary",
			serializer:   jsonSerializer,
			deserializer: binSerializer,
		},
		{
			name:         "Binary2JSON",
			serializer:   binSerializer,
			deserializer: jsonSerializer,
		},
		{
			name:         "Binary2Binary",
			serializer:   binSerializer,
			deserializer: binSerializer,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := tc.serializer.Serialize(d)
			require.NoError(t, err)
			var singlePtr durationpb.Duration
			err = tc.deserializer.Deserialize(b, &singlePtr)
			require.NoError(t, err)
			require.Equal(t, time.Minute, singlePtr.AsDuration())
			var doublePtr *durationpb.Duration
			err = tc.deserializer.Deserialize(b, &doublePtr)
			require.NoError(t, err)
			require.Equal(t, time.Minute, doublePtr.AsDuration())
		})
	}
}
