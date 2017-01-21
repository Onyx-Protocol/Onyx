package testutil

import "reflect"

// DeepEqual is similar to reflect.DeepEqual, but treats
// nil and []T{} as equal.
// (Note, it's also a more naive implementation that doesn't detect
// cycles).
func DeepEqual(x, y interface{}) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)
	return deepValueEqual(vx, vy)
}

func deepValueEqual(x, y reflect.Value) bool {
	if isNilish(x) && isNilish(y) {
		return true
	}
	if !x.IsValid() {
		return !y.IsValid()
	}
	if !y.IsValid() {
		return false
	}

	tx := x.Type()
	ty := y.Type()
	if tx != ty {
		return false
	}
	switch tx.Kind() {
	case reflect.Bool:
		return x.Bool() == y.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return x.Int() == y.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return x.Uint() == y.Uint()

	case reflect.Float32, reflect.Float64:
		return x.Float() == y.Float()

	case reflect.Complex64, reflect.Complex128:
		return x.Complex() == y.Complex()

	case reflect.String:
		return x.String() == y.String()

	case reflect.Array:
		for i := 0; i < tx.Len(); i++ {
			if !deepValueEqual(x.Index(i), y.Index(i)) {
				return false
			}
		}
		return true

	case reflect.Slice:
		ttx := tx.Elem()
		tty := ty.Elem()
		if ttx != tty {
			return false
		}
		if x.Len() != y.Len() {
			return false
		}
		for i := 0; i < x.Len(); i++ {
			if !deepValueEqual(x.Index(i), y.Index(i)) {
				return false
			}
		}
		return true

	case reflect.Interface:
		if x.IsNil() {
			return y.IsNil()
		}
		if y.IsNil() {
			return false
		}
		return deepValueEqual(x.Elem(), y.Elem())

	case reflect.Ptr:
		if x.Pointer() == y.Pointer() {
			return true
		}
		return deepValueEqual(x.Elem(), y.Elem())

	case reflect.Struct:
		for i := 0; i < tx.NumField(); i++ {
			if !deepValueEqual(x.Field(i), y.Field(i)) {
				return false
			}
		}
		return true

	case reflect.Map:
		if x.Pointer() == y.Pointer() {
			return true
		}
		if x.Len() != y.Len() {
			return false
		}
		for _, k := range x.MapKeys() {
			if !deepValueEqual(x.MapIndex(k), y.MapIndex(k)) {
				return false
			}
		}
		return true

	case reflect.Func:
		return x.IsNil() && y.IsNil()
	}
	return false
}

func isNilish(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Type().Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr:
		return v.IsNil()

	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	}
	return false
}
