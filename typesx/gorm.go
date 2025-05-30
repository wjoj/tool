package typesx

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type Time struct {
	time.Time
}

func (j *Time) Scan(value any) error {
	v, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("time conversion error: %T", value)
	}

	*j = Time{
		Time: v,
	}
	return nil
}

func (j Time) Value() (driver.Value, error) {
	if j.IsZero() {
		return nil, nil
	}
	return j.Local().Format(time.DateTime), nil
}

func (s Time) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(`0`), nil
	}
	return []byte(strconv.FormatInt(s.Local().Unix(), 10)), nil
}

func (s *Time) UnmarshalJSON(data []byte) error {
	if s == nil {
		return errors.New("unmarshaling JSON value is empty(time)")
	}
	t, err := time.Parse("2006-01-02 15:04:05", string(data))
	if err != nil {
		return err
	}
	*s = Time{Time: t}
	return nil
}

type Date struct {
	time.Time
}

func NewDataString(str string) (Date, error) {
	t, err := time.Parse(time.DateOnly, str)
	return Date{
		Time: t,
	}, err
}

func (j *Date) Scan(value any) error {
	v, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("date to time conversion error: %T", value)
	}

	*j = Date{
		Time: v,
	}
	return nil
}

func (j Date) Value() (driver.Value, error) {
	if j.IsZero() {
		return nil, nil
	}
	return j.Local().Format(time.DateOnly), nil
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
	t, err := time.Parse("2006-01-02", string(data))
	if err != nil {
		return err
	}
	*s = Date{Time: t}
	return nil
}
