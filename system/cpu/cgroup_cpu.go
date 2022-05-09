package cpu

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pscpu "github.com/shirou/gopsutil/v3/cpu"
	"github.com/wjoj/tool/base"
)

const (
	cpuTicks  = 100
	cpuFields = 8
)

type CgroupCpu struct {
	frequency uint64
	quota     float64 // cpu数量
	cores     uint64  // 可用cpu

	preSystemUsage uint64
	preTotalUsage  uint64
	usage          uint64
}

// Info cpu info.
type Info struct {
	Frequency uint64
	Quota     float64
}

func (c *CgroupCpu) Info() Info {
	return Info{
		Frequency: c.frequency,
		Quota:     c.quota,
	}
}

func (c *CgroupCpu) Usage() (uint64, error) {

	var (
		total  uint64
		system uint64
		u      uint64
	)
	total, err := totalUsage()
	if err != nil {
		return 0, err
	}
	system, err = systemCpuUsage()
	if err != nil {
		return 0, err
	}
	if system != c.preSystemUsage {
		u = uint64(float64((total-c.preTotalUsage)*c.cores*1e3) / (float64(system-c.preSystemUsage) * c.quota))
	}
	c.preSystemUsage = system
	c.preTotalUsage = total
	return u, nil
}

// NewCgroupCpu return linux cpu
func NewCgroupCpu() (*CgroupCpu, error) {

	cores, err := pscpu.Counts(true)
	if err != nil || cores == 0 {
		perCpuUsage, err := PerCpuUsage()
		if err != nil {
			return nil, fmt.Errorf("fail to PerCpuUsage(), e:%s", err.Error())
		}

		cores = len(perCpuUsage)
	}

	cpus, err := Cpus()
	if err != nil {
		return nil, fmt.Errorf("fail to Cpus(), e:%s", err.Error())
	}

	quota := float64(len(cpus))

	cpuQuota, err := CpuQuota()
	if err == nil && cpuQuota != -1 {
		period, err := CpuPeriod()
		if err != nil {
			return nil, fmt.Errorf("fail to CpuPeriod(), e:%s", err.Error())
		}

		// logic cpus from time period
		limit := float64(cpuQuota) / float64(period)
		if limit < quota {
			quota = limit
		}

	}

	// The maximum run frequency supported by CPU hardware
	maxFreq := cpuMaxFreq()

	preSystemUsage, err := systemCpuUsage()
	if err != nil {
		return nil, fmt.Errorf("fail to systemCpuUsage(), e:%s", err.Error())
	}

	totalU, err := totalUsage()
	if err != nil {
		return nil, fmt.Errorf("fail to totalUsage(), e:%s", err.Error())
	}

	cpu := &CgroupCpu{
		frequency:      maxFreq,
		quota:          quota,
		cores:          uint64(cores),
		preSystemUsage: preSystemUsage,
		preTotalUsage:  totalU,
	}
	return cpu, nil
}

func totalUsage() (uint64, error) {
	cg, err := CurrentCgroup()
	if err != nil {
		return 0, err
	}

	return cg.CpuAcctAllUsage()
}

func systemCpuUsage() (uint64, error) {
	lines, err := base.ReadLines("/proc/stat")
	if err != nil {
		return 0, err
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {

			// https://man7.org/linux/man-pages/man5/proc.5.html
			if len(fields) < cpuFields {
				return 0, fmt.Errorf("bad format of cpu stats")
			}

			var totalClockTicks uint64
			for _, i := range fields[1:cpuFields] {
				v, err := ParseUint(i)
				if err != nil {
					return 0, err
				}

				totalClockTicks += v
			}

			return (totalClockTicks * uint64(time.Second)) / cpuTicks, nil
		}
	}

	return 0, errors.New("bad stats format")
}

func cpuMaxFreq() uint64 {
	feq := cpuFreq()
	data, err := base.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq")
	if err != nil {
		return feq
	}
	// override the max freq from /proc/cpuinfo
	cfeq, err := ParseUint(data)
	if err == nil {
		feq = cfeq
	}
	return feq
}

func cpuFreq() uint64 {
	lines, err := base.ReadLines("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		if key == "cpu MHz" || key == "clock" {
			// treat this as the fallback value, thus we ignore error
			if t, err := strconv.ParseFloat(strings.Replace(value, "MHz", "", 1), 64); err == nil {
				return uint64(t * 1000.0 * 1000.0)
			}
		}
	}
	return 0
}

func CpuPeriod() (uint64, error) {
	cg, err := CurrentCgroup()
	if err != nil {
		return 0, err
	}
	return cg.CpuPeriodUS()
}

func CpuQuota() (int64, error) {
	cg, err := CurrentCgroup()
	if err != nil {
		return 0, err
	}
	return cg.CpuQuotaUS()
}

func Cpus() ([]uint64, error) {
	cg, err := CurrentCgroup()
	if err != nil {
		return nil, err
	}

	return cg.Cpus()
}

func PerCpuUsage() ([]uint64, error) {
	cg, err := CurrentCgroup()
	if err != nil {
		return nil, err
	}

	return cg.CpuAcctPerUsage()
}
