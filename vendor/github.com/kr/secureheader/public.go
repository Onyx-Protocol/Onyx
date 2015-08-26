// +build !heroku

package secureheader

// Assume that the web server might be exposed to the public
// internet, and that clients might send all sorts of crazy values
// in HTTP headers. Therefore, don't believe X-Forwarded-Proto.
// Furthermore, even if X-Forwarded-Proto is known to be accurate,
// conservatively treat an unencrypted last hop as sufficient to
// mean the request is unsafe.
const defaultUseForwardedProto = false
