/*

Command appenv reads and writes Chain app environment variables
(aka config vars).

Usage

	appenv [flags] [name|name=value...]

With no arguments, it prints all config vars and their values.
Given just a name, it prints the value for that config var.
Given one or more name=value arguments, it merges them into
the stack's config.

Flag -a specifies the app to access.
The default is "api".

Flag -t specifies the target to access.
The default is the value of $USER.

Flag -r specifies the release to access.
The default is "next", which means to get
and set values that will be used for future releases.
Values can only be set on the "next" release;
past releases are readonly.

Format

The app config vars are stored in S3 as a JSON array of strings.
When chainbot makes a release, it translates the JSON document
to a bash script that can be sourced to initialize the environment.

*/
package main
