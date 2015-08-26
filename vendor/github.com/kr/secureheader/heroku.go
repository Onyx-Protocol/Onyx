// +build heroku

package secureheader

// Heroku dynos sit behind a proxy. Trust that the proxy sets
// X-Forwarded-Proto correctly and that eavesdropping won't happen
// on the last hop.
const defaultUseForwardedProto = true
