package utils

import "time"

type Duration string

func (d Duration) ParseDuration() (time.Duration, error) {
	return time.ParseDuration(string(d))
}
