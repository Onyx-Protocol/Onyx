/*

Command mdfmt canonicalizes the formatting of a Markdown document.

Usage:

	mdfmt <inp.md >out.md

For now, only whitespace is canonicalized:
	- Trailing whitespace is removed from each line;
	- Multiple blank lines are collapsed down to one;
	- Before each heading line (that begins with #), a blank line is added for each heading level being unwound.

For example, given this input:

	# Heading 1a


	some text
	## Heading 2

	more text

	# Heading 1b

This program will produce:

	# Heading 1a

	some text

	## Heading 2

	more text


	# Heading 1b

*/
package main
