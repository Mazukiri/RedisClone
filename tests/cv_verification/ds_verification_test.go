package data_structure_test

import (
	"fmt"
	"math/rand"
	"github.com/Mazukiri/RedisClone/internal/data_structure"
	"testing"
)

// Helper to create a dummy SkipList if needed or use existing one
// Assuming data_structure.SkipList usage

func BenchmarkSkipListSearch(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Size=%d", n), func(b *testing.B) {
			sl := data_structure.CreateSkiplist()
			// Populate
			for i := 0; i < n; i++ {
				sl.Insert(float64(i), fmt.Sprintf("val%d", i))
			}

			// Measure random lookup
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target := float64(rand.Intn(n))
				sl.GetRank(target, fmt.Sprintf("val%d", int(target)))
			}
		})
	}
}
