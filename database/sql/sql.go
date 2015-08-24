package sql

import (
	"database/sql"
	"errors"
	"reflect"
)

var errCollectType = errors.New("destination must be pointer to a slice of structs")

// Collect assembles the results of rows into dest.
// It will close rows if any error is encountered.
func Collect(rows *sql.Rows, dest interface{}) error {
	defer rows.Close()
	destVal := reflect.ValueOf(dest)
	t, err := makeType(dest)
	if err != nil {
		return err
	}
	for rows.Next() {
		d, args := makeArgs(t)
		err := rows.Scan(args...)
		if err != nil {
			return err
		}
		appendVal(destVal, d)
	}
	return rows.Err()
}

func makeType(dest interface{}) (reflect.Type, error) {
	typ := reflect.TypeOf(dest)
	if typ.Kind() != reflect.Ptr {
		return nil, errCollectType
	}
	if typ.Elem().Kind() != reflect.Slice {
		return nil, errCollectType
	}
	if typ.Elem().Elem().Kind() != reflect.Struct {
		return nil, errCollectType
	}
	return typ.Elem().Elem(), nil
}

func makeArgs(t reflect.Type) (v reflect.Value, args []interface{}) {
	v = reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		val := v.Elem().Field(i)
		args = append(args, val.Addr().Interface())
	}
	return v, args
}

func appendVal(dest, v reflect.Value) {
	slice := dest.Elem()
	slice = reflect.Append(slice, v.Elem())
	dest.Elem().Set(slice)
}
