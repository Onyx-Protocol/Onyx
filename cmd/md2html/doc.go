/*

Command md2html converts a directory of markdown files to html. For each
markdown file in a given directory, md2html will parse that markdown to html
and write the resulting html to a file of the same name in the destination
directory.

Usage

	md2html source-dir destination-dir

If there is a file named layout.html which contains `{{Body}}`
in the source-dir, md2html use that data to wrap the html output
of the converted markdown.
*/
package main
