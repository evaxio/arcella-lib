package wcache

import (
	"sync"
	"testing"
)

type TestStruct struct {
	TestInt   int
	TestStr   string
	TestFloat float64
}

func Test01(t *testing.T) {
	c := NewCache[int, TestStruct]()
	for i := 0; i < 10_001; i++ {
		c.Set(i, TestStruct{TestInt: i, TestStr: "test", TestFloat: float64(i)})
	}
	v1, ok := c.Get(1000)
	if !ok {
		t.Fatalf("Get(1000) not found")
	}
	if v1.TestInt != 1000 || v1.TestStr != "test" || v1.TestFloat != 1000 {
		t.Errorf("Get(1000) = %+v, want {1000 test 1000}", v1)
	}
	v2, ok := c.Get(10_000)
	if !ok {
		t.Fatalf("Get(10_000) not found")
	}
	if v2.TestInt != 10_000 || v2.TestStr != "test" || v2.TestFloat != 10_000 {
		t.Errorf("Get(10_000) = %+v, want {10000 test 10000}", v2)
	}
	if _, ok := c.Get(-1); ok {
		t.Errorf("Get(-1) = found, want not found")
	}
}

func TestSetGetOverwriteDelete(t *testing.T) {
	c := NewCache[string, int]()
	if _, ok := c.Get("k"); ok {
		t.Fatalf("Get(k) on empty cache = found, want not found")
	}
	c.Set("k", 1)
	v, ok := c.Get("k")
	if !ok || *v != 1 {
		t.Fatalf("Get(k) = %v, %v; want 1, true", v, ok)
	}
	c.Set("k", 2)
	v, ok = c.Get("k")
	if !ok || *v != 2 {
		t.Fatalf("Get(k) after overwrite = %v, %v; want 2, true", v, ok)
	}
	c.Del("k")
	if _, ok := c.Get("k"); ok {
		t.Errorf("Get(k) after Del = found, want not found")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewCache[int, int]()
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				k := g*500 + i
				c.Set(k, k)
				if v, ok := c.Get(k); !ok || *v != k {
					t.Errorf("Get(%d) = %v, %v; want %d, true", k, v, ok, k)
					return
				}
				if i%3 == 0 {
					c.Del(k)
				}
			}
		}(g)
	}
	wg.Wait()
}
