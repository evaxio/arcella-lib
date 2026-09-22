package fail2ban

import (
	log "log/slog"
	"sync"
	"time"
)

type Suspended struct {
	items     map[string]time.Time
	banPeriod time.Duration
	rwMutex   sync.RWMutex
	lastSweep time.Time
}

func NewSuspended(period time.Duration) *Suspended {
	return &Suspended{banPeriod: period, items: make(map[string]time.Time)}
}

func (s *Suspended) Add(ip string) {
	s.rwMutex.Lock()
	s.items[ip] = time.Now().Add(s.banPeriod)
	s.rwMutex.Unlock()
}

func (s *Suspended) Is(ip string) bool {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	now := time.Now()
	if len(s.items) > suspectedEvictThreshold && time.Since(s.lastSweep) > evictSweepInterval {
		for itemIP, t := range s.items {
			if t.Before(now) {
				delete(s.items, itemIP)
			}
		}
		s.lastSweep = now
	}
	t, ok := s.items[ip]
	if !ok { // record not found
		return false
	}
	if t.Before(now) { // unban
		delete(s.items, ip)
		log.Debug("UNBAN", log.String("ip", ip))
		return false
	}
	return true // yes, still Suspended
}
