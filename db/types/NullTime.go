package types

import (
	"database/sql/driver"
	"errors"
	"reflect"
	"time"
)

type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}
	var s string
	switch v := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = v, true
		return nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return errors.New("cannot scan " + reflect.TypeOf(value).String() + " into NullTime")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		nt.Time, nt.Valid = t, true
		return nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		nt.Time, nt.Valid = t, true
		return nil
	}
	return errors.New("cannot scan " + reflect.TypeOf(value).String() + " into NullTime")
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}
