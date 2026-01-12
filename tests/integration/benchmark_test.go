package integration

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkSet(b *testing.B) {
	rdb := SetupClient()
	defer rdb.Close()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			key := fmt.Sprintf("bench_set_key_%d", i)
			rdb.Set(ctx, key, "value", 0)
		}
	})
}

func BenchmarkGet(b *testing.B) {
	rdb := SetupClient()
	defer rdb.Close()
	ctx := context.Background()

	// Pre-populate
	rdb.Set(ctx, "bench_get_key", "value", 0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rdb.Get(ctx, "bench_get_key")
		}
	})
}
