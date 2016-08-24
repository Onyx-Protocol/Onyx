/*

Package httpjson creates HTTP handlers to map request
and response formats onto Go function signatures.
The request body is decoded as a JSON text
into an arbitrary value and the function's return value
is encoded as a JSON text for the response body.
The function's signature determines the types of the
input and output values.

For example, the handler for a function with signature

  func(struct{ FavColor, Birthday string })

would read the JSON request body into a variable
of type struct{ FavColor, Birthday string }, then call
the function.

The allowed elements of a function signature are:

  parameters:
  Context (optional)
  request body (optional)

  return values:
  response body (optional)
  error (optional)

All elements are optional.
Thus, the smallest possible function signature is

  func()

If the function returns a non-nil error,
the handler will call the error function provided
in its constructor.
Otherwise, the handler will write the return value
as JSON text to the response body.
If the return type is omitted, the handler will send
a default response value.

*/
package httpjson
