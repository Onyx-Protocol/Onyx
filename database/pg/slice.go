package pg

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Byteas [][]byte

func (a *Byteas) Scan(val interface{}) error {
	panic("unimplemented")
}

// Value encodes the slice of byte slices as a psql array of bytea.
// Go value: Byteas{{'f', 'o', 'o'}, {'b', 'a', 'r'}}
// Postgres syntax: {\\x666f6f,\\x626172}
func (a Byteas) Value() (driver.Value, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, b := range a {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, `\\x%x`, b)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

type Strings []string

// Scan currently only handles simple values exlcuding commas and quotes.
func (a *Strings) Scan(val interface{}) error {
	*a = nil
	if val == nil {
		return nil
	}
	s, ok := val.([]byte)
	if !ok {
		return errors.New("invalid interface for Scan")
	}
	for _, el := range bytes.Split(s[1:len(s)-1], []byte(",")) {
		if len(el) == 0 {
			continue
		}
		if el[0] == '"' && el[len(el)-1] == '"' && len(el) >= 2 {
			*a = append(*a, string(el[1:len(el)-1]))
		} else if el[0] == '"' || el[len(el)-1] == '"' {
			return fmt.Errorf("unsupported syntax or error in %s", s)
		} else {
			*a = append(*a, string(el))
		}
	}
	return nil
}

func (a Strings) Value() (driver.Value, error) {
	var val []byte
	val = append(val, '{')
	for i, s := range a {
		if i > 0 {
			val = append(val, ',')
		}
		b := []byte(s)
		val = append(val, '"')
		for _, c := range b {
			switch c {
			case '"', '\\':
				val = append(val, '\\', c)
			default:
				val = append(val, c)
			}
		}
		val = append(val, '"')
	}
	return append(val, '}'), nil
}

type NullStrings []sql.NullString

func (a *NullStrings) Scan(val interface{}) error {
	panic("unimplemented")
}

func (a NullStrings) Value() (driver.Value, error) {
	var val []byte
	val = append(val, '{')
	for i, s := range a {
		if i > 0 {
			val = append(val, ',')
		}
		if !s.Valid {
			val = append(val, "NULL"...)
			continue
		}

		b := []byte(s.String)
		val = append(val, '"')
		for _, c := range b {
			switch c {
			case '"', '\\':
				val = append(val, '\\', c)
			default:
				val = append(val, c)
			}
		}
		val = append(val, '"')
	}
	return append(val, '}'), nil
}

type Uint32s []uint32

func (a *Uint32s) Scan(val interface{}) error {
	*a = nil
	if val == nil {
		return nil
	}
	b, ok := val.([]byte)
	if !ok {
		return errors.New("invalid interface for Scan")
	}
	s := string(b)
	for _, el := range strings.Split(s[1:len(s)-1], ",") {
		if len(el) == 0 {
			continue
		}
		n, err := strconv.ParseUint(el, 10, 32)
		if err != nil {
			return err
		}
		*a = append(*a, uint32(n))
	}
	return nil
}

func (a Uint32s) Value() (driver.Value, error) {
	var val []byte
	val = append(val, '{')
	for i, ui := range a {
		if i > 0 {
			val = append(val, ',')
		}
		val = strconv.AppendUint(val, uint64(ui), 10)
	}
	return append(val, '}'), nil
}

type Uint64s []uint64

func (a *Uint64s) Scan(val interface{}) error {
	*a = nil
	if val == nil {
		return nil
	}
	b, ok := val.([]byte)
	if !ok {
		return errors.New("invalid interface for Scan")
	}
	s := string(b)
	for _, el := range strings.Split(s[1:len(s)-1], ",") {
		if len(el) == 0 {
			continue
		}
		n, err := strconv.ParseUint(el, 10, 64)
		if err != nil {
			return err
		}
		*a = append(*a, n)
	}
	return nil
}

func (a Uint64s) Value() (driver.Value, error) {
	var val []byte
	val = append(val, '{')
	for i, ui := range a {
		if i > 0 {
			val = append(val, ',')
		}
		val = strconv.AppendUint(val, ui, 10)
	}
	return append(val, '}'), nil
}

type Int64s []int64

func (a *Int64s) Scan(val interface{}) error {
	*a = nil
	if val == nil {
		return nil
	}
	b, ok := val.([]byte)
	if !ok {
		return errors.New("invalid interface for Scan")
	}
	s := string(b)
	for _, el := range strings.Split(s[1:len(s)-1], ",") {
		if len(el) == 0 {
			continue
		}
		n, err := strconv.ParseInt(el, 10, 64)
		if err != nil {
			return err
		}
		*a = append(*a, n)
	}
	return nil
}

func (a Int64s) Value() (driver.Value, error) {
	var val []byte
	val = append(val, '{')
	for i, i64 := range a {
		if i > 0 {
			val = append(val, ',')
		}
		val = strconv.AppendInt(val, i64, 10)
	}
	return append(val, '}'), nil
}

type Int32s []int32

func (a *Int32s) Scan(val interface{}) error {
	*a = nil
	if val == nil {
		return nil
	}
	b, ok := val.([]byte)
	if !ok {
		return errors.New("invalid interface for Scan")
	}
	s := string(b)
	for _, el := range strings.Split(s[1:len(s)-1], ",") {
		if len(el) == 0 {
			continue
		}
		n, err := strconv.ParseInt(el, 10, 32)
		if err != nil {
			return err
		}
		*a = append(*a, int32(n))
	}
	return nil
}

func (a Int32s) Value() (driver.Value, error) {
	var val []byte
	val = append(val, '{')
	for i, i32 := range a {
		if i > 0 {
			val = append(val, ',')
		}
		val = strconv.AppendInt(val, int64(i32), 10)
	}
	return append(val, '}'), nil
}

type Bools []bool

func (a *Bools) Scan(val interface{}) error {
	*a = nil
	if val == nil {
		return nil
	}
	b, ok := val.([]byte)
	if !ok {
		return errors.New("invalid interface for Scan")
	}
	s := string(b)
	for _, el := range strings.Split(s[1:len(s)-1], ",") {
		if len(el) == 0 {
			continue
		}
		n := el == "t"
		*a = append(*a, n)
	}
	return nil
}

func (a Bools) Value() (driver.Value, error) {
	var val []byte
	val = append(val, '{')
	for i, b := range a {
		if i > 0 {
			val = append(val, ',')
		}
		if b {
			val = append(val, 't')
			continue
		}
		val = append(val, 'f')
	}
	return append(val, '}'), nil
}
