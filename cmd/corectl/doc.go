/*

Command corectl provides miscellaneous control functions for a Chain Core.

The database connection can be configured using the DB_URL environment
variable; the default is to connect to the "core" database on localhost.

    corectl adduser [email] [password] [role]

Adduser creates user accounts. The standard method of adding user accounts via
an invite flow can be inconvenient for development purposes, so this tool
provides an easy command-line alternative. It should be called with three
command-line arguments, an email address, a password, and a role.

    corectl genesis

Genesis creates a genesis block.

    corectl boot

Boot bootstraps the database to a minimal functional state:

    user
    auth token
    admin node

*/
package main
