package memcachettl

import (
	"axgit.vixiv.ru/snake/arcella-lib/utils"
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

type TestStruct struct {
	TestInt   int
	TestStr   string
	TestFloat float64
}

func Test01(t *testing.T) {
	c := NewCache()
	for i := 0; i < 10_001; i++ {
		c.Set(utils.IntToStr(i), TestStruct{TestInt: i, TestStr: "test", TestFloat: float64(i)}, time.Microsecond)
	}
	time.Sleep(time.Millisecond)
	if _, ok := c.Get("1000"); ok {
		t.Errorf("Get(1000) returned an expired item, want not found")
	}
	if _, ok := c.Get("10000"); ok {
		t.Errorf("Get(10000) returned an expired item, want not found")
	}
	if _, ok := c.Get("missing"); ok {
		t.Errorf("Get(missing) = found, want not found")
	}
}

func Test02(t *testing.T) {
	c := NewCache()
	for i := 0; i < 10_001; i++ {
		c.Set(utils.IntToStr(i), TestStruct{TestInt: i, TestStr: "test", TestFloat: float64(i)}, time.Microsecond)
	}
	if n := c.Size(); n != 10_001 {
		t.Errorf("Size = %d, want 10_001", n)
	}
	for i := 0; i < 10_001; i++ {
		c.Set(utils.IntToStr(i), TestStruct{TestInt: i, TestStr: "test", TestFloat: float64(i)}, time.Microsecond)
	}
	if n := c.Size(); n != 10_001 {
		t.Errorf("Size after overwrite = %d, want 10_001", n)
	}
	time.Sleep(time.Millisecond)
	c.CheckTTL()
	if n := c.Size(); n != 0 {
		t.Errorf("Size after CheckTTL = %d, want 0", n)
	}
}

func TestSetGetDeleteOverwrite(t *testing.T) {
	c := NewCache()
	if _, ok := c.Get("k"); ok {
		t.Fatalf("Get(k) on empty cache = found, want not found")
	}
	c.Set("k", TestStruct{TestInt: 1}, time.Hour)
	v, ok := c.Get("k")
	if !ok {
		t.Fatalf("Get(k) not found after Set")
	}
	if s := v.(TestStruct); s.TestInt != 1 {
		t.Errorf("Get(k) = %+v, want TestInt 1", s)
	}
	c.Set("k", TestStruct{TestInt: 2}, time.Hour)
	v, ok = c.Get("k")
	if !ok {
		t.Fatalf("Get(k) not found after overwrite")
	}
	if s := v.(TestStruct); s.TestInt != 2 {
		t.Errorf("Get(k) after overwrite = %+v, want TestInt 2", s)
	}
	c.Delete("k")
	if _, ok := c.Get("k"); ok {
		t.Errorf("Get(k) after Delete = found, want not found")
	}
	if n := c.Size(); n != 0 {
		t.Errorf("Size = %d, want 0", n)
	}
}

func TestGetExpiredNotReturned(t *testing.T) {
	c := NewCache()
	c.Set("k", TestStruct{TestInt: 42}, 30*time.Millisecond)
	v, ok := c.Get("k")
	if !ok {
		t.Fatalf("Get(k) not found before expiry")
	}
	if s := v.(TestStruct); s.TestInt != 42 {
		t.Errorf("Get(k) = %+v, want TestInt 42", s)
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Errorf("Get(k) returned an expired item, want not found")
	}
}

func TestCheckTTLEvicts(t *testing.T) {
	c := NewCache()
	c.Set("expired", "v1", 20*time.Millisecond)
	c.Set("live", "v2", time.Hour)
	time.Sleep(40 * time.Millisecond)
	c.CheckTTL()
	if _, ok := c.Get("expired"); ok {
		t.Errorf("CheckTTL did not evict the expired key")
	}
	if v, ok := c.Get("live"); !ok || v != "v2" {
		t.Errorf("CheckTTL evicted a live key: v=%v ok=%v", v, ok)
	}
}

func TestSizeDoesNotEvict(t *testing.T) {
	c := NewCache()
	for i := 0; i < 100; i++ {
		c.Set(utils.IntToStr(i), i, 20*time.Millisecond)
	}
	time.Sleep(40 * time.Millisecond)
	if n := c.Size(); n != 100 {
		t.Errorf("Size = %d, want 100 (Size must not trigger eviction)", n)
	}
	c.CheckTTL()
	if n := c.Size(); n != 0 {
		t.Errorf("Size after CheckTTL = %d, want 0", n)
	}
}

func TestGetAllCallbackCanMutate(t *testing.T) {
	c := NewCache()
	for i := 0; i < 10; i++ {
		c.Set(utils.IntToStr(i), i, time.Hour)
	}
	var seen int
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.GetAll(func(key string, item CacheItem) {
			seen++
			c.Delete(key)
			c.Set("extra", item.Value, time.Hour)
		})
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("GetAll deadlocked while the callback mutates the cache")
	}
	if seen != 10 {
		t.Errorf("callback invoked %d times, want 10", seen)
	}
	if _, ok := c.Get("extra"); !ok {
		t.Errorf("Set inside the GetAll callback did not take effect")
	}
}

func TestStopIdempotentAndStopsReaper(t *testing.T) {
	base := stableGoroutines(t)

	c := NewCache()
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if n := stableGoroutines(t); n != base+1 {
		t.Fatalf("goroutines after Start = %d, want %d (one reaper)", n, base+1)
	}
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop must be idempotent: %v", err)
	}
	if n := stableGoroutines(t); n != base {
		t.Errorf("goroutines after Stop = %d, want %d (reaper leaked)", n, base)
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				k := utils.IntToStr(i)
				c.Set(k, i, time.Hour)
				c.Get(k)
				if i%3 == 0 {
					c.Delete(k)
				}
				c.Size()
			}
		}()
	}
	wg.Wait()
	c.CheckTTL()
}

// stableGoroutines waits until the goroutine count stops changing.
func stableGoroutines(t *testing.T) int {
	t.Helper()
	n := runtime.NumGoroutine()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		time.Sleep(20 * time.Millisecond)
		m := runtime.NumGoroutine()
		if m == n {
			return n
		}
		n = m
	}
	return n
}
