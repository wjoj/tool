package system

import (
	"sync/atomic"
	"time"

	"github.com/wjoj/tool/system/cpu"
)

// CPU is cpu stat usage.
type CPU interface {
	Usage() (u uint64, e error)
	Info() cpu.Info
}

// Stat cpu stat.
type Stat struct {
	Usage uint64 // cpu use ratio.
}

const (
	interval time.Duration = time.Millisecond * 500
)

var (
	stats CPU
	usage uint64
)

func init() {
	var err error
	stats, err = cpu.NewCgroupCpu()
	if err != nil {
		//fmt.Println("fail to NewCgroupCpu(), e:", err.Error())
		stats, err = cpu.NewPsutilCPU(interval)
		if err != nil {
			panic("fail to NewPsutilCPU(), e:" + err.Error())
		}
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			u, err := stats.Usage()
			if err == nil && u != 0 {
				atomic.StoreUint64(&usage, u)
			}
		}

	}()

}

// LoadStat read cpu stat.
func LoadStat(stat *Stat) {
	stat.Usage = atomic.LoadUint64(&usage)
}

// GetInfo get cpu info.
func GetInfo() cpu.Info {
	return stats.Info()
}
