package field

import (
	"testing"

	"github.com/cimartinez3/iso8583/encoding"
	"github.com/cimartinez3/iso8583/prefix"
	"github.com/stretchr/testify/require"
)

func TestBinaryField(t *testing.T) {
	spec := &Spec{
		Length:      10,
		Description: "Field",
		Enc:         encoding.Binary,
		Pref:        prefix.Binary.Fixed,
	}

	in := []byte("1234567890")

	t.Run("Pack returns binary data", func(t *testing.T) {
		bin := NewBinary(spec)
		bin.SetBytes(in)

		packed, err := bin.Pack()

		require.NoError(t, err)
		require.Equal(t, in, packed)
	})

	t.Run("String returns binary data encoded in HEX", func(t *testing.T) {
		bin := NewBinary(spec)
		bin.Value = in

		str, err := bin.String()

		require.NoError(t, err)
		require.Equal(t, "31323334353637383930", str)
	})

	t.Run("Unpack returns binary data", func(t *testing.T) {
		bin := NewBinary(spec)

		n, err := bin.Unpack(in)

		require.NoError(t, err)
		require.Equal(t, len(in), n)
		require.Equal(t, in, bin.Value)
	})

	t.Run("SetData sets data to the field", func(t *testing.T) {
		bin := NewBinary(spec)
		bin.SetData(NewBinaryValue(in))

		packed, err := bin.Pack()

		require.NoError(t, err)
		require.Equal(t, in, packed)
	})

	t.Run("Unpack sets data to data value", func(t *testing.T) {
		bin := NewBinary(spec)
		data := NewBinaryValue([]byte{})
		bin.SetData(data)

		n, err := bin.Unpack(in)

		require.NoError(t, err)
		require.Equal(t, len(in), n)
		require.Equal(t, in, data.Value)
	})
}
