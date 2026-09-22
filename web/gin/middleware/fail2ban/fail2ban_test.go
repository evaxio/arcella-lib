package fail2ban

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSuspendedBanAndUnban(t *testing.T) {
	s := NewSuspended(50 * time.Millisecond)
	if s.Is("1.1.1.1") {
		t.Fatal("unknown ip must not be suspended")
	}
	s.Add("1.1.1.1")
	if !s.Is("1.1.1.1") {
		t.Fatal("ip must be suspended after Add")
	}
	time.Sleep(80 * time.Millisecond)
	if s.Is("1.1.1.1") {
		t.Fatal("ip must be unbanned after the ban period")
	}
}

func TestFail2BanBannedAfterThreshold(t *testing.T) {
	fb := newFail2Ban(&ConfigFail2Ban{
		WatchPeriod: 2 * time.Second,
		WatchCount:  3,
		BanPeriod:   50 * time.Millisecond,
	})
	ip := "9.9.9.9"
	fb.processStatusNotFound(ip)
	fb.processStatusNotFound(ip)
	if fb.suspended.Is(ip) {
		t.Fatal("ip must not be banned before the threshold")
	}
	fb.processStatusNotFound(ip)
	if !fb.suspended.Is(ip) {
		t.Fatal("ip must be banned after the threshold")
	}
	time.Sleep(80 * time.Millisecond)
	if fb.suspended.Is(ip) {
		t.Fatal("ban must expire after the ban period")
	}
}

func TestSuspectedEviction(t *testing.T) {
	s := NewSuspected(20 * time.Millisecond)
	for i := 0; i < suspectedEvictThreshold+10; i++ {
		s.Incr(fmt.Sprintf("10.%d.%d.%d", i/65536, (i/256)%256, i%256))
	}
	time.Sleep(50 * time.Millisecond)
	s.mu.Lock()
	s.lastSweep = time.Time{}
	s.mu.Unlock()
	s.Incr("192.168.0.1") // triggers the stale entries sweep
	s.mu.Lock()
	n := len(s.items)
	s.mu.Unlock()
	if n != 1 {
		t.Fatalf("stale entries not evicted: %d items left, want 1", n)
	}
}

func TestFail2BanConcurrent(t *testing.T) {
	suspended := NewSuspended(time.Second)
	suspected := NewSuspected(time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		ip := fmt.Sprintf("10.0.%d.%d", i/256, i%256)
		wg.Add(3)
		go func() {
			defer wg.Done()
			suspended.Add(ip)
		}()
		go func() {
			defer wg.Done()
			suspended.Is(ip)
		}()
		go func() {
			defer wg.Done()
			suspected.Incr(ip)
		}()
	}
	wg.Wait()
}
