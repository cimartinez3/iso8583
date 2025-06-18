package describe

import (
	"bytes"
	"testing"

	"github.com/cimartinez3/iso8583"
	"github.com/cimartinez3/iso8583/specs"
	"github.com/stretchr/testify/require"
)

func TestDescribe(t *testing.T) {
	message := iso8583.NewMessage(specs.Spec87ASCII)

	require.NotPanics(t, func() {
		Message(bytes.NewBuffer([]byte{}), message)
	})
}
