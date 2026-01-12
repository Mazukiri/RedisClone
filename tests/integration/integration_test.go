package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

// SetupClient creates a new Redis client connecting to the MemKV server
func SetupClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:8082",
		Password: "", // no password set
		DB:       0,  // use default DB
		Protocol: 2,
	})
}

func TestBasicOperations(t *testing.T) {
	rdb := SetupClient()
	defer rdb.Close()

	// PING
	pong, err := rdb.Ping(ctx).Result()
	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)

	// SET
	err = rdb.Set(ctx, "key", "value", 0).Err()
	assert.NoError(t, err)

	// GET
	val, err := rdb.Get(ctx, "key").Result()
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	// GET missing
	_, err = rdb.Get(ctx, "missing_key").Result()
	assert.Error(t, err)

	// EXPIRE
	set := rdb.Expire(ctx, "key", 10*time.Second).Val()
	assert.True(t, set)

	// TTL
	ttl, err := rdb.TTL(ctx, "key").Result()
	assert.NoError(t, err)
	assert.True(t, ttl > 0)
}

func TestZSetOperations(t *testing.T) {
	rdb := SetupClient()
	defer rdb.Close()
	key := "myzset_test"

	// Cleanup
	rdb.Del(ctx, key)

	// ZADD
	added, err := rdb.ZAdd(ctx, key, redis.Z{Score: 1, Member: "one"}, redis.Z{Score: 2, Member: "two"}).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), added)

	// ZSCORE
	score, err := rdb.ZScore(ctx, key, "one").Result()
	assert.NoError(t, err)
	assert.Equal(t, 1.0, score)

	// ZRANK
	rank, err := rdb.ZRank(ctx, key, "two").Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rank)

	// ZREM
	removed, err := rdb.ZRem(ctx, key, "one").Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), removed)
}

func TestSetOperations(t *testing.T) {
	rdb := SetupClient()
	defer rdb.Close()
	key := "myset_test"
	rdb.Del(ctx, key)

	// SADD
	added, err := rdb.SAdd(ctx, key, "a", "b", "c").Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), added)

	// SISMEMBER
	isMember, err := rdb.SIsMember(ctx, key, "a").Result()
	assert.NoError(t, err)
	assert.True(t, isMember)

	isMember, err = rdb.SIsMember(ctx, key, "d").Result()
	assert.NoError(t, err)
	assert.False(t, isMember)

	// SMEMBERS
	members, err := rdb.SMembers(ctx, key).Result()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "b", "c"}, members)
}

func TestConcurrentAccess(t *testing.T) {
	rdb := SetupClient()
	defer rdb.Close()
	key := "concurrent_counter"
	rdb.Set(ctx, key, 0, 0)

	var wg sync.WaitGroup
	workers := 20
	requests := 50

	// Simulating multiple clients incrementing a key (using raw INCR if implemented, else just SETs)
	// Since INCR might be on TODO or string specific, let's use SET random keys to stress IO
	
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			localClient := SetupClient()
			defer localClient.Close()

			for j := 0; j < requests; j++ {
				k := fmt.Sprintf("key_%d_%d", id, j)
				err := localClient.Set(ctx, k, "val", 0).Err()
				assert.NoError(t, err)
				
				_, err = localClient.Get(ctx, k).Result()
				assert.NoError(t, err)
			}
		}(i)
	}
	wg.Wait()
}
