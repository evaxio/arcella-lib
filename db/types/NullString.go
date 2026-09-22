package types

import (
	"database/sql/driver"
	"errors"
	"reflect"
)

type NullString struct {
	Val   string
	Valid bool
}

// Scan implements the Scanner interface.
func (nt *NullString) Scan(value interface{}) error {
	if value == nil {
		nt.Val, nt.Valid = "", false
		return nil
	}
	switch v := value.(type) {
	case string:
		nt.Val, nt.Valid = v, true
	case []byte:
		nt.Val, nt.Valid = string(v), true
	default:
		return errors.New("cannot scan " + reflect.TypeOf(value).String() + " into NullString")
	}
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullString) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Val, nil
}

func (nt NullString) ValidValue() string {
	if !nt.Valid {
		return ""
	}
	return nt.Val
}
