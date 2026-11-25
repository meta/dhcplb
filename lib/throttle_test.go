/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func Test_ThrottleArgs(t *testing.T) {
	// Test invalid cache size
	_, err := NewThrottle(-1, 128, 128)
	if err == nil {
		t.Fatalf("Should return error on negative cache size")
	}
}

func Test_ThrottleOff(t *testing.T) {
	// Test non throttling option
	throttle, err := NewThrottle(128, 64, -1)
	if err != nil {
		t.Fatalf("Error creating a throttle: %s", err)
	}

	for i := 0; i < 1000; i++ {
		// Test one key multiple requests
		if ok, err := throttle.OK("my_key"); !ok {
			t.Fatalf("Throttling disabled, shouldn't throttle requests: %s", err)
		}

		// Test multiple keys
		key := fmt.Sprintf("my_key_%d", i)
		if ok, err := throttle.OK(key); !ok {
			t.Fatalf("Throttling disabled, shouldn't throttle new items: %s", err)
		}
	}
}

func Test_ThrottleCacheRateOff(t *testing.T) {
	throttle, err := NewThrottle(128, -1, 64)
	if err != nil {
		t.Fatalf("Error creating a throttle: %s", err)
	}

	for i := 0; i < 1000; i++ {
		// Test multiple keys
		key := fmt.Sprintf("my_key_%d", i)
		if ok, err := throttle.OK(key); !ok {
			t.Fatalf("Cache rate limiting is disabled, shouldn't throttle new items: %s", err)
		}
	}
}

func Test_ThrottleLRU(t *testing.T) {
	const cacheSize = 128
	const cacheRate = 128 // per sec
	const connRate = 1    // per sec

	sleepTime := time.Duration(math.Ceil((1000. / cacheRate) / 2))
	loopCount := cacheSize * 2

	throttle, err := NewThrottle(cacheSize, cacheRate, connRate)
	if err != nil {
		t.Fatalf("Error creating a throttle: %v", err)
	}
	for i := 0; i < loopCount; i++ {
		key := fmt.Sprintf("my_key_%d", i)

		throttle.OK(key)

		time.Sleep(sleepTime * time.Millisecond)
	}
	if throttle.len() != cacheSize {
		t.Fatalf("Throttle LRU size is wrong - expected value: %d", cacheSize)
	}
}

func Test_ThrottleSingleConnection(t *testing.T) {
	const cacheSize = 1
	const cacheRate = 1 // per sec
	const connRate = 64 // per sec
	const key = "test_key"

	sleepDuration := time.Duration(math.Ceil(1000. / connRate))
	loopCount := connRate

	throttle, err := NewThrottle(cacheSize, cacheRate, connRate)
	if err != nil {
		t.Fatalf("Error creating a throttle: %v", err)
	}
	for i := 0; i < loopCount; i++ {
		if ok, err := throttle.OK(key); !ok {
			t.Fatalf("Throttling failed for single connection: %s", err)
		}

		time.Sleep(sleepDuration * time.Millisecond)
	}
}

func Test_ThrottleSingleConnectionFail(t *testing.T) {
	const cacheSize = 1
	const cacheRate = 1 // per sec
	const connRate = 64 // per sec
	const key = "test_key"

	sleepTime := time.Duration((1000 / connRate) / 3)
	loopCount := connRate * 3

	throttle, err := NewThrottle(cacheSize, cacheRate, connRate)
	if err != nil {
		t.Fatalf("Error creating a throttle: %v", err)
	}
	for i := 0; i < loopCount; i++ {
		if ok, _ := throttle.OK(key); !ok {
			return
		}

		time.Sleep(sleepTime * time.Millisecond)
	}

	t.Fatalf("Throttling didn't work for single connection!")
}

func Test_ThrottleCacheRate(t *testing.T) {
	const cacheSize = 1024
	const cacheRate = 64
	const connRate = 1

	sleepTime := time.Duration(math.Ceil(1000. / cacheRate))
	loopCount := cacheRate

	throttle, err := NewThrottle(cacheSize, cacheRate, connRate)
	if err != nil {
		t.Fatalf("Error creating a throttle: %v", err)
	}
	for i := 0; i < loopCount; i++ {
		key := fmt.Sprintf("my_key_%d", i)

		if ok, err := throttle.OK(key); !ok {
			t.Fatalf("Throttling failed for cache rate limiting: %s", err)
		}

		time.Sleep(sleepTime * time.Millisecond)
	}
}

func Test_ThrottleCacheRateFail(t *testing.T) {
	const cacheSize = 1024
	const cacheRate = 64
	const connRate = 1

	sleepTime := time.Duration((1000 / cacheRate) / 3)
	loopCount := cacheRate * 3

	throttle, err := NewThrottle(cacheSize, cacheRate, connRate)
	if err != nil {
		t.Fatalf("Error creating a throttle: %v", err)
	}
	for i := 0; i < loopCount; i++ {
		key := fmt.Sprintf("my_key_%d", i)

		if ok, _ := throttle.OK(key); !ok {
			return
		}

		time.Sleep(sleepTime * time.Millisecond)
	}

	t.Fatalf("Throttling didn't work for cache rate limiting")
}

func Test_ThrottleSetRate(t *testing.T) {
	const key = "test_key"

	throttle, err := NewThrottle(16, -1, 1)
	if err != nil {
		t.Fatalf("Error creating a throttle: %v", err)
	}
	if ok, err := throttle.OK(key); !ok {
		t.Fatalf("First request should not be throttled: %s", err)
	}
	if ok, _ := throttle.OK(key); ok {
		t.Fatalf("Second request should be throttled at 1 req/sec")
	}

	// key is already in the cache, it must get the new rate
	throttle.setRate(1000)
	time.Sleep(10 * time.Millisecond)
	if ok, err := throttle.OK(key); !ok {
		t.Fatalf("New rate was not applied to cached key: %s", err)
	}

	throttle.setRate(1)
	throttle.OK(key)
	if ok, _ := throttle.OK(key); ok {
		t.Fatalf("Lower rate was not applied to cached key")
	}
}

func Test_ThrottleSetRateConcurrent(t *testing.T) {
	throttle, err := NewThrottle(16, -1, 1)
	if err != nil {
		t.Fatalf("Error creating a throttle: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			throttle.setRate(i%2 + 1)
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		throttle.OK("test_key")
	}
	<-done
}
