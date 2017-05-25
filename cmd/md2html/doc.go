/*

Command md2html recursively converts a directory of markdown files to html.
For each markdown file in the current directory, md2html will convert
the markdown to html and write the resulting html to a file of
the same name in the destination directory.

If there is a file named layout.html which contains `{{.Body}}`
in the current directory, md2html use that data to wrap the html output
of the converted markdown.

Usage

	md2html build DEST_PATH


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

	$ md2html serve [port]

Code Interpolation

Markdown files may contain $code references that will pull in
source code from other files on the filesystem into the markdown file.
This interpolation happens prior to converting the markdown into HTML.

To use a $code reference in markdown:

	$code abs-path-to-file [snippet name]

If the snippet name is omitted, the entire file is interpolated.

To define a snippet in a source file:

	//snippet [name]
	// ... code
	//endsnippet

For example, assume we have the following file at `/src/x.go`

	package main

	import "fmt"

	func main() {
		add(1, 2)
		fmt.Println("hello world")
	}

	//snippet add
	func add(x, y int) {
		return x + y
	}
	//endsnippet

And assume we have the following markdown file:

	# Go Example

	$code /src/x.go

md2html will produce the following interpolated markdown file:

	# Go Example

	```
	package main

	import "fmt"

	func main() {
		add(1, 2)
		fmt.Println("hello world")
	}

	func add(x, y int) {
		return x + y
	}
	```

Assume we have another markdown file:

	# Go Example

	$code /src/x.go add

md2html will produce the following interpolated markdown file:

	# Go Example

	```
	func add(x, y int) {
		return x + y
	}
	```

Sidenotes

To wrap a portion of the document in a sidenote tag
`<div class="sidenote"> ... </div>` use these tags:

	[sidenote]

	...

	[/sidenote]

Make sure that each tag occupies its own paragraph per
markdown rules, otherwise processor will not recognize them.


Local Links

This tool converts local links ending with `.md` by stripping
that extension, so the resulting HTML link points to a formatted
page instead of a raw markdown file.


*/
package main
