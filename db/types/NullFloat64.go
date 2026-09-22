package types

import (
	"database/sql/driver"
	"errors"
	"reflect"
	"strconv"
)

type NullFloat64 struct {
	value float64
	Valid bool // Valid is true if Float64 is not NULL
}

// Scan implements the Scanner interface.
func (nt *NullFloat64) Scan(value interface{}) error {
	if value == nil {
		nt.value, nt.Valid = 0, false
		return nil
	}
	switch v := value.(type) {
	case float64:
		nt.value, nt.Valid = v, true
	case int64:
		nt.value, nt.Valid = float64(v), true
	case []byte:
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return err
		}
		nt.value, nt.Valid = f, true
	default:
		return errors.New("cannot scan " + reflect.TypeOf(value).String() + " into NullFloat64")
	}
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullFloat64) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.value, nil
}
