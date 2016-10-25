/*

Command hex encodes to or from hexadecimal.

Usage:

	hex [-d] [-n len]

Flag -d decodes.
In this mode, hex reads hexadecimal characters from stdin,
skipping over SPC, TAB, LF, and CR,
and writes the decoded binary bytes to stdout.
It is an error for the input to include any other characters
or to contain an odd number of hex characters.
Without -d, it encodes binary to hex.

Flag -n applies only to encoding.
It inserts a newline
after every n characters of output (n/2 bytes of input),
and at the end.
If n is odd, it is rounded down to the nearest even.
If n < 2, it suppresses all newlines from the output.
The default is 2,147,483,647 (2**31 - 1).

*/
package main
