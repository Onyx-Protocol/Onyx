/*

Command report executes its argument as a child process
and reports the results to S3.

It uploads the full combined output (stdout and stderr)
plus the exit code (if nonzero)
to the S3 object "run/[started at]-[name]",
then updates a summary table of past runs
at 'results.json' and 'index.html'.

It also writes the time of the last run
in RFC 3339 format in S3 object 'lastrun'.

All output of this command goes to S3.
The only exception is error messages for failures accessing S3,
which are written to stderr.

*/
package main
