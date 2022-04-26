package tool

import "time"

func Time(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func Date(t time.Time) string {
	return t.Format("2006-01-02")
}

func Clock(t time.Time) string {
	return t.Format("15:04:05")
}
