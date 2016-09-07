package pg

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

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
