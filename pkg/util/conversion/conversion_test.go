package conversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUInt32ToBinaryCompressed(t *testing.T) {
	testCases := []struct {
		in  uint32
		out []byte
	}{
		{99, []byte{'\x63'}},
		{12388, []byte{'\x30', '\x64'}},
		{0, []byte{'\x00'}},
	}

	for _, tc := range testCases {
		out, _ := UInt32ToBinaryCompressed(tc.in)
		assert.Equal(t, tc.out, out)
	}
}
