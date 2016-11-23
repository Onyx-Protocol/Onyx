package pg

import (
	"context"
	"reflect"

	"chain/errors"
)

var ErrBadRequest = errors.New("bad request")

// The type of "error"
var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

// ForQueryRows encapsulates a lot of boilerplate when making db queries.
// Call it like this:
//
//   err = ForQueryRows(ctx, db, query, queryArg1, queryArg2, ..., func(scanVar1 type1, scanVar2 type2, ...) {
//     ...process a row from the result...
//   })
//
// This is equivalent to:
//
//   rows, err = db.Query(ctx, query, queryArg1, queryArg2, ...)
//   if err != nil {
//     return err
//   }
//   defer rows.Close()
//   for rows.Next() {
//     var (
//       scanVar1 type1
//       scanVar2 type2
//     )
//     err = rows.Scan(&scanVar1, &scanVar2, ...)
//     if err != nil {
//       return err
//     }
//     ...process a row from the result...
//   }
//   if err = rows.Err(); err != nil {
//     return err
//   }
//
// The callback is invoked once for each row in the result.  The
// number and types of parameters to the callback must match the
// values to be scanned with rows.Scan.  The space for the callback's
// arguments is not reused between calls.  The callback may return a
// single error-type value.  If any invocation yields a non-nil
// result, ForQueryRows will abort and return it.
func ForQueryRows(ctx context.Context, db DB, query string, args ...interface{}) error {
	if len(args) == 0 {
		return errors.Wrap(ErrBadRequest, "too few arguments")
	}

	fnArg := args[len(args)-1]
	queryArgs := args[:len(args)-1]

	fnType := reflect.TypeOf(fnArg)
	if fnType.Kind() != reflect.Func {
		return errors.Wrap(ErrBadRequest, "fn arg not a function")
	}
	if fnType.NumOut() > 1 {
		return errors.Wrap(ErrBadRequest, "fn arg must return 0 values or 1")
	}
	if fnType.NumOut() == 1 && !fnType.Out(0).Implements(errorInterface) {
		return errors.Wrap(ErrBadRequest, "fn arg return type must be error")
	}

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return errors.Wrap(err, "query")
	}
	defer rows.Close()

	fnVal := reflect.ValueOf(fnArg)

	argPtrVals := make([]reflect.Value, 0, fnType.NumIn())
	scanArgs := make([]interface{}, 0, fnType.NumIn())
	fnArgs := make([]reflect.Value, 0, fnType.NumIn())

	for rows.Next() {
		argPtrVals = argPtrVals[:0]
		scanArgs = scanArgs[:0]
		fnArgs = fnArgs[:0]
		for i := 0; i < fnType.NumIn(); i++ {
			argType := fnType.In(i)
			argPtrVal := reflect.New(argType)
			argPtrVals = append(argPtrVals, argPtrVal)
			scanArgs = append(scanArgs, argPtrVal.Interface())
		}
		err = rows.Scan(scanArgs...)
		if err != nil {
			return errors.Wrap(err, "scan")
		}
		for _, argPtrVal := range argPtrVals {
			fnArgs = append(fnArgs, argPtrVal.Elem())
		}
		res := fnVal.Call(fnArgs)
		if fnType.NumOut() == 1 && !res[0].IsNil() {
			return errors.Wrap(res[0].Interface().(error), "callback")
		}
	}

	return errors.Wrap(rows.Err(), "end scan")
}
