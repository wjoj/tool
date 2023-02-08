package db

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

type Time struct {
	time.Time
}

func (s Time) Value() (driver.Value, error) {
	if s.IsZero() {
		return nil, nil
	}
	return s.Local().Format("2006-01-02 15:04:05"), nil
}

func (s *Time) Scan(value interface{}) error {
	v, ok := value.(time.Time)
	if ok {
		*s = Time{Time: v}
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", value)
}

func (s Time) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + s.Local().Format("2006-01-02 15:04:05") + `"`), nil
}

func (s *Time) UnmarshalJSON(data []byte) error {
	if s == nil {
		return errors.New("null point exception")
	}
	t, err := time.Parse(string(data), "2006-01-02 15:04:05")
	if err != nil {
		return err
	}
	*s = Time{Time: t}
	return nil
}

type Date struct {
	time.Time
}

func (s Date) Value() (driver.Value, error) {
	if s.IsZero() {
		return nil, nil
	}
	return s.Local().Format("2006-01-02"), nil
}

func (s *Date) Scan(value interface{}) error {
	v, ok := value.(time.Time)
	if ok {
		*s = Date{Time: v}
		return nil
	}
	return fmt.Errorf("can not convert %v to date", value)
}

func (s Date) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + s.Local().Format("2006-01-02") + `"`), nil
}

func (s *Date) UnmarshalJSON(data []byte) error {
	if s == nil {
		return errors.New("null point exception")
	}
	t, err := time.Parse(string(data), "2006-01-02")
	if err != nil {
		return err
	}
	*s = Date{Time: t}
	return nil
}
