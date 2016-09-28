/*

Command md2html recursively converts a directory of markdown files to html.
For each markdown file in the current directory, md2html will convert
the markdown to html and write the resulting html to a file of
the same name in the destination directory.

If there is a file named layout.html which contains `{{Body}}`
in the current directory, md2html use that data to wrap the html output
of the converted markdown.

Usage

	md2html [destination]


	destination - address or directory

	If the destination is an address (e.g. :8080) then md2html will start
	a web server and serve file from the current directory on that port.
	If the file request contains an extension, the corresponding file will be
	served unprocessed. If the file request does not contain an extension,
	the corresponding file will be converted from markdown into HTML before
	it is served.

	If the destination is a directory, md2html will recursively copy all files
	from . to destination -- converting .md files into HTML along the way.

Example

	Start a server in the current directory:
	$ md2html

	Copy all files --and convert .md to HTML-- in the current directory and
	write them to ~/site:
	$ md2html ~/site

*/
package main
