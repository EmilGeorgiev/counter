package benchmark

import (
	"encoding/binary"
	"strconv"
	"testing"
)

func BenchmarkBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, 1)
		_ = binary.BigEndian.Uint64(buf)
	}
}

func BenchmarkString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = strconv.Atoi("1")
	}
}
