package core_test

import (
	"github.com/Mazukiri/RedisClone/internal/core"
	"testing"
)

func BenchmarkParseCmd(b *testing.B) {
	data := []byte("*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$7\r\nmyvalue\r\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.ParseCmd(data)
	}
}

func BenchmarkEncode(b *testing.B) {
	data := []interface{}{"SET", "mykey", "myvalue"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.Encode(data, false)
	}
}
