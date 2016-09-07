/*
Package errors implements a basic error wrapping pattern, so that errors can be
annotated with additional information without losing the original error.

Example:

	import "chain/errors"

	func query() error {
		err := pq.Exec("SELECT...")
		if err != nil {
			return errors.Wrap(err, "select query failed")
		}

		err = pq.Exec("INSERT...")
		if err != nil {
			return errors.Wrap(err, "insert query failed")
		}

		return nil
	}

	func main() {
		err := query()
		if _, ok := errors.Root(err).(sql.ErrNoRows); ok {
			log.Println("There were no results")
			return
		} else if err != nil {
			log.Println(err)
			return
		}

		log.Println("success")
	}

When to wrap errors

Errors should be wrapped with additional messages when the context is ambiguous.
This includes when the error could arise in multiple locations in the same
function, when the error is very common and likely to appear at different points
in the call tree (e.g., JSON serialization errors), or when you need specific
parameters alongside the original error message.

Error handling best practices

Errors are part of a function's interface. If you expect the caller to perform
conditional error handling, you should document the errors returned by your
function in a function comment, and include it as part of your unit tests.

Be disciplined about validating user input. Programs should draw a very clear
distinction between user errors and internal errors.

Avoid redundant error logging. If you return an error, assume it will be logged
higher up the call stack. For a given project, choose an appropriate layer to
handle error logging.
*/
package errors
