package cpu

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/wjoj/tool/base"
)

const cgroupRootDir = "/sys/fs/cgroup"

type cgroup struct {
	cgroupSet map[string]string
}

// CpuAcctAllUsage CPUACCT.USAGE Reports the total CPU time (in nanoseconds) used
// by all tasks in a cgroup (including all tasks in their descendents).
// This file can be written with a value of 0 to reset statistics.
func (c *cgroup) CpuAcctAllUsage() (uint64, error) {
	text, err := base.ReadText(path.Join(c.cgroupSet["cpuacct"], "cpuacct.usage")) // ex: 1838571117964229
	if err != nil {
		return 0, err
	}
	return ParseUint(text)
}

func (c *cgroup) CpuAcctPerUsage() ([]uint64, error) {
	text, err := base.ReadText(path.Join(c.cgroupSet["cpuacct"], "cpuacct.usage_percpu")) // ex: 924274335970490 914283397203835
	if err != nil {
		return nil, err
	}

	var usage []uint64
	for _, v := range strings.Fields(text) {
		u, err := ParseUint(v)
		if err != nil {
			return nil, err
		}

		usage = append(usage, u)
	}

	return usage, nil

}

// Cpus cpu nodes current cgroup can used
func (c *cgroup) Cpus() ([]uint64, error) {
	text, err := base.ReadText(path.Join(c.cgroupSet["cpuset"], "cpuset.cpus")) // ex: 0-1
	if err != nil {
		return nil, err
	}
	return ParseUints(text)
}

// CpuQuotaUS CPU.CFS_QUOTA_US is the amount of CPU time available during this period,
// which defaults to -1, meaning unlimited
func (c *cgroup) CpuQuotaUS() (int64, error) {
	data, err := base.ReadText(path.Join(c.cgroupSet["cpu"], "cpu.cfs_quota_us")) // ex: -1
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(data, 10, 64)
}

// CpuPeriodUS CPU.CFS_PERIOD_US is the time period, which defaults to 100000, 100 milliseconds(micro seconds)
func (c *cgroup) CpuPeriodUS() (uint64, error) {
	data, err := base.ReadText(path.Join(c.cgroupSet["cpu"], "cpu.cfs_period_us")) // ex: 100000
	if err != nil {
		return 0, err
	}

	return ParseUint(data)

}

func CurrentCgroup() (cg *cgroup, err error) {

	pid := os.Getpid()
	cgFile := fmt.Sprintf("/proc/%d/cgroup", pid)

	fp, err := os.Open(cgFile)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	var lines []string
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}

	cgroupSet := make(map[string]string)

	// every line should be liked:
	// 2:cpu,cpuacct:/user.slice/user-0.slice/session-52279.scope
	for _, line := range lines {

		cols := strings.Split(line, ":")
		if len(cols) != 3 {
			return nil, fmt.Errorf("invalid cgroup line: %s", line)
		}

		subsys := cols[1]

		// get cpu stat
		if !strings.Contains(subsys, "cpu") {
			continue
		}

		// https://man7.org/linux/man-pages/man7/cgroups.7.html
		// comma-separated list of controllers for cgroup version 1
		fields := strings.Split(subsys, ",")
		for _, field := range fields {
			cgroupSet[field] = path.Join(cgroupRootDir, field)
		}

	}

	return &cgroup{cgroupSet: cgroupSet}, nil

}
