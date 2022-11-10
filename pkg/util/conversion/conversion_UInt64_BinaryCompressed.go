package conversion

import (
	"encoding/binary"
	"fmt"
	"net"
)

func UInt64ToBinaryCompressed(i uint64) ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	for idx := 0; idx < 8; idx++ {
		if b[idx] != 0 {
			return b[idx:], nil
		}
	}
	return []byte{'\x00'}, nil
}

func BinaryCompressedToUint64(bytes []byte) uint64 {
	buff := make([]byte, 8)
	offset := 8 - len(bytes)
	for idx, b := range bytes {
		buff[offset+idx] = b
	}
	return binary.BigEndian.Uint64(buff)
}
