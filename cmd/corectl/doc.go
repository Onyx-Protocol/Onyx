/*

Command corectl provides miscellaneous control functions for a Chain Core.

The database connection can be configured using the DB_URL environment
variable; the default is to connect to the "core" database on localhost.

    corectl init [quorum] [key...]

Init creates an initial block. Its consensus program contains the given keys
and requires quorum signatures. A quorum size of 0 is ok.

*/
package main
