/*

Command perfdash is a web server that serves a performance dashboard for Chain Core.
Environment variable LISTEN determines its listen address (default :8080).

Its index page uses query param "baseurl" to connect to a running Chain Core.
The default is https://localhost:1999/.

Examples

Connect to a local Chain Core running on the default port.

	http://localhost:8080/

Connect to the testnet generator Chain Core.

	http://localhost:8080/?baseurl=https://testnet.chain.com:8443/

*/
package main
