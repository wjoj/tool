package util

import (
	"log"
	"sync/atomic"
	"time"
)

//const statInterval = time.Minute
const statInterval = time.Second

type Stat struct {
	name         string
	hit          uint64
	miss         uint64
	sizeCallback func() int
}

func NewStat(name string, sizeCallback func() int) *Stat {
	st := &Stat{
		name:         name,
		sizeCallback: sizeCallback,
	}
	go st.report()

	return st
}

// Hit hit counter++
func (s *Stat) Hit() {
	atomic.AddUint64(&s.hit, 1)
}

// Miss missed counter++
func (s *Stat) Miss() {
	atomic.AddUint64(&s.miss, 1)
}

// CurrentMinute return hit and missed counter in a minute
func (s *Stat) CurrentMinute() (hit, miss uint64) {
	hit = atomic.LoadUint64(&s.hit)
	miss = atomic.LoadUint64(&s.miss)
	return hit, miss
}

func (s *Stat) report() {
	ticker := time.NewTicker(statInterval)
	defer ticker.Stop()

	for range ticker.C {
		hit := atomic.SwapUint64(&s.hit, 0)
		miss := atomic.SwapUint64(&s.miss, 0)
		total := hit + miss
		if total == 0 {
			log.Printf("cache(%s) - continue", s.name)
			continue
		}

		percent := float32(hit) / float32(total)
		log.Printf("cache(%s) - qpm: %d, hit_ratio: %.1f%%, elements: %d, hit: %d, miss: %d",
			s.name, total, percent*100, s.sizeCallback(), hit, miss)
	}
}
