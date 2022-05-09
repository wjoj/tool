package cpu

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseUint(s string) (uint64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		if err.(*strconv.NumError).Err == strconv.ErrRange {
			return 0, nil
		}

		return 0, fmt.Errorf("cgroup: bad int format: %s", s)
	}

	if v < 0 {
		return 0, nil
	}

	return uint64(v), nil
}

func ParseUints(val string) ([]uint64, error) {
	if val == "" {
		return nil, nil
	}

	ints := make(map[uint64]struct{})
	cols := strings.Split(val, ",")
	for _, r := range cols {
		if strings.Contains(r, "-") {
			fields := strings.SplitN(r, "-", 2)
			min, err := ParseUint(fields[0])
			if err != nil {
				return nil, fmt.Errorf("cgroup: bad int list format: %s", val)
			}

			max, err := ParseUint(fields[1])
			if err != nil {
				return nil, fmt.Errorf("cgroup: bad int list format: %s", val)
			}

			if max < min {
				return nil, fmt.Errorf("cgroup: bad int list format: %s", val)
			}

			for i := min; i <= max; i++ {
				ints[i] = struct{}{}
			}
		} else {
			v, err := ParseUint(r)
			if err != nil {
				return nil, err
			}

			ints[v] = struct{}{}
		}
	}

	var sets []uint64
	for k := range ints {
		sets = append(sets, k)
	}

	return sets, nil
}
