//+build insecure-disable-https-redirect

package main

/*

The secureheader package redirects requests made with http to the
equivalent https URL. This file exposes a build tag to turn
that functionality off. The functionality will stay on by default. This
will be useful for chain core developer edition. Users will be able to
connect to a chain core without a needing a TLS cert.

*/

func init() {
	httpsRedirect = false
}
