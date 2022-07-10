package conversion

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TimeStringType string

func (t TimeStringType) Duration() TimeDurationType {
	str := string(t)
	if strings.HasSuffix(str, "n") {
		number, _ := strconv.Atoi(str[:len(str)-1])
		return TimeDurationType(number) * TimeDurationType(time.Nanosecond)
	} else if strings.HasSuffix(str, "µ") {
		number, _ := strconv.Atoi(str[:len(str)-1])
		return TimeDurationType(number) * TimeDurationType(time.Microsecond)
	} else if strings.HasSuffix(str, "ms") {
		number, _ := strconv.Atoi(str[:len(str)-2])
		return TimeDurationType(number) * TimeDurationType(time.Millisecond)
	} else if strings.HasSuffix(str, "s") {
		number, _ := strconv.Atoi(str[:len(str)-1])
		return TimeDurationType(number) * TimeDurationType(time.Second)
	} else if strings.HasSuffix(str, "m") {
		number, _ := strconv.Atoi(str[:len(str)-1])
		return TimeDurationType(number) * TimeDurationType(time.Minute)
	} else if strings.HasSuffix(str, "h") {
		number, _ := strconv.Atoi(str[:len(str)-1])
		return TimeDurationType(number) * TimeDurationType(time.Hour)
	} else if strings.HasSuffix(str, "h") {
		number, _ := strconv.Atoi(str[:len(str)-1])
		return TimeDurationType(number) * TimeDurationType(time.Hour)
	} else {
		number, _ := strconv.Atoi(str)
		return TimeDurationType(number) * TimeDurationType(time.Second)
	}
}

type TimeDurationType time.Duration

func (t TimeDurationType) String() string {
	dur := time.Duration(t)
	if time.Nanosecond*1000 > dur {
		return fmt.Sprintf("%dn", dur)
	} else if (time.Microsecond * 1000) > dur {
		return fmt.Sprintf("%dµ%dn", dur/time.Microsecond, dur%time.Microsecond)
	} else if (time.Millisecond * 1000) > dur {
		return fmt.Sprintf("%dms%dµ%dn", dur/(time.Millisecond), (dur%time.Millisecond)/1000, (dur%(time.Millisecond))%1000)
	} else if (time.Second * 60) > dur {
		ms := t % TimeDurationType(time.Second)
		return fmt.Sprintf("%ds%dms%dµ%dn", t/TimeDurationType(time.Second), ms/1000, (ms%1000)/1000, (ms%1000)%1000)
	} else if (time.Minute * 60) > dur {
		m := t % TimeDurationType(time.Minute)
		ms := m % 60
		return fmt.Sprintf("%dm%ds%dms%dµ%dn", t/TimeDurationType(time.Minute), m/60, ms, (ms%1000)/1000, (ms%1000)%1000)
	}
	return ""
}
