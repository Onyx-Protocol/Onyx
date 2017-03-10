package testutil

import (
	"reflect"
	"unsafe"
)

type visit struct {
	a1, a2 unsafe.Pointer
	typ    reflect.Type
}

// DeepEqual is similar to reflect.DeepEqual, but treats nil as equal
// to empty maps and slices. Some of the implementation is cribbed
// from Go's reflect package.
func DeepEqual(x, y interface{}) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)
	return deepValueEqual(vx, vy, make(map[visit]bool))
}

func deepValueEqual(x, y reflect.Value, visited map[visit]bool) bool {
	if isEmpty(x) && isEmpty(y) {
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
	case reflect.Array, reflect.Map, reflect.Slice, reflect.Struct:
		if x.CanAddr() && y.CanAddr() {
			a1 := unsafe.Pointer(x.UnsafeAddr())
			a2 := unsafe.Pointer(y.UnsafeAddr())
			if uintptr(a1) > uintptr(a2) {
				// Canonicalize order to reduce number of entries in visited.
				// Assumes non-moving garbage collector.
				a1, a2 = a2, a1
			}
			v := visit{a1, a2, tx}
			if visited[v] {
				return true
			}
			visited[v] = true
		}
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
			if !deepValueEqual(x.Index(i), y.Index(i), visited) {
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
			if !deepValueEqual(x.Index(i), y.Index(i), visited) {
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
		return deepValueEqual(x.Elem(), y.Elem(), visited)

	case reflect.Ptr:
		if x.Pointer() == y.Pointer() {
			return true
		}
		return deepValueEqual(x.Elem(), y.Elem(), visited)

	case reflect.Struct:
		for i := 0; i < tx.NumField(); i++ {
			if !deepValueEqual(x.Field(i), y.Field(i), visited) {
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
			if !deepValueEqual(x.MapIndex(k), y.MapIndex(k), visited) {
				return false
			}
		}
		return true

	case reflect.Func:
		return x.IsNil() && y.IsNil()
	}
	return false
}

func isEmpty(v reflect.Value) bool {
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
