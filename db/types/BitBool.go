// https://stackoverflow.com/questions/47535543/mysqls-bit-type-maps-to-which-go-type
package types

import (
	"database/sql/driver"
	"errors"
)

// BitBool is an implementation of a bool for the MySQL type BIT(1).
// This type allows you to avoid wasting an entire byte for MySQL's boolean type TINYINT.
type BitBool bool

// Scan implements the sql.Scanner interface,
// and turns the bitfield incoming from MySQL into a BitBool
func (b *BitBool) Scan(src interface{}) error {
	if src == nil {
		*b = false
		return nil
	}
	v, ok := src.([]byte)
	if !ok {
		return errors.New("bad []byte type assertion")
	}
	if len(v) == 0 {
		return errors.New("empty []byte")
	}
	*b = v[0] == 1
	return nil
}

// Value implements the driver.Valuer interface,
// and turns the BitBool into a bitfield (BIT(1)) for MySQL storage.
func (b BitBool) Value() (driver.Value, error) {
	if b {
		return []byte{1}, nil
	} else {
		return []byte{0}, nil
	}
}
