package fail2ban

import (
	"sync"
	"time"
)

type SuspectedItem struct {
	Count     int
	WatchTime time.Time
}

func (si *SuspectedItem) TooLate() bool {
	return si.WatchTime.Before(time.Now())
}

/////////////////////////////////////////////////////////////////////////////////

type Suspected struct {
	items       map[string]SuspectedItem
	mu          sync.Mutex
	lastSweep   time.Time
	watchPeriod time.Duration
}

func NewSuspected(watchPeriod time.Duration) *Suspected {
	return &Suspected{items: make(map[string]SuspectedItem), watchPeriod: watchPeriod}
}

// suspectedEvictThreshold triggers a sweep of stale entries when the map grows
// beyond it, keeping memory bounded under IP scans.
const suspectedEvictThreshold = 1024

const evictSweepInterval = 30 * time.Second

func (s *Suspected) Incr(ip string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if len(s.items) > suspectedEvictThreshold && time.Since(s.lastSweep) > evictSweepInterval {
		for itemIP, item := range s.items {
			if item.WatchTime.Before(now) {
				delete(s.items, itemIP)
			}
		}
		s.lastSweep = now
	}
	item, ok := s.items[ip]
	if !ok {
		item = SuspectedItem{Count: 1, WatchTime: now.Add(s.watchPeriod)}
		s.items[ip] = item
		return item.Count
	}
	if item.WatchTime.Before(now) {
		delete(s.items, ip)
		return 0
	}
	item.Count++
	s.items[ip] = item
	return item.Count
}

func (s *Suspected) Del(ip string) {
	s.mu.Lock()
	delete(s.items, ip)
	s.mu.Unlock()
}
