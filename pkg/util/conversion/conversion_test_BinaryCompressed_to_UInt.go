package conversion

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBinaryCompressedToUint64(t *testing.T) {
	nums := []uint64{0, 64, 1024, 10240}
	for _, n := range nums {
		iCompressed, _ := UInt64ToBinaryCompressed(n)
		iDecompress := BinaryCompressedToUint64(iCompressed)
		assert.Equal(t, n, iDecompress)
	}
}

func TestBinaryCompressedToUint16(t *testing.T) {
	nums := []uint64{0, 6, 32, 64}
	for _, n := range nums {
		iCompressed, _ := UInt64ToBinaryCompressed(n)
		iDecompress := BinaryCompressedToUint64(iCompressed)
		assert.Equal(t, n, iDecompress)
	}
}
