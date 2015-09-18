/*

Package httpjson provides an HTTP handler to map HTTP request
and response formats onto Go function signatures.
Elements of the request path are converted into string parameters,
the request body is decoded as a JSON text
into an arbitrary value, and the function's return value
is encoded as a JSON text for the response body.
The function's signature determines the types of the
input and output values.

Each function is registered as a handler using a pattern
to match request paths. Each pattern has one or more labels,
which are placeholder elements that match arbitrary text
in the request path.
See package chain/net/http/pat for more details.

For example, a function with signature

  func(string, struct{ FavColor, Birthday string })

would take a string from the request path
(as determined by the pattern used to register the function in ServeMux)
and read the JSON request body into a variable
of type struct{ FavColor, Birthday string }.

The allowed elements of a function signature are:

  parameters:
  Context (optional)
  strings, one for each label in the pattern
  request body (optional)

  return values:
  response body (optional)
  error (optional)

All elements are optional except the path strings taken from
pattern labels, and there may be zero labels in the pattern.
Thus, the smallest possible function signature is

  func()

If the function returns a non-nil error,
the handler will call the error function provided
in its constructor.
Otherwise, the handler will write the return value
as JSON text to the reponse body.
If the response value is omitted, the handler will send
a default response value.

*/
package httpjson
