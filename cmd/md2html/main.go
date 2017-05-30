package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/russross/blackfriday"
)

var extToLang = map[string]string{
	"java": "Java",
	"rb":   "Ruby",
	"js":   "Node",
}

type command struct {
	f func([]string)
}

var commands = map[string]*command{
	"serve": {serve},
	"build": {convert},
}

func main() {
	if len(os.Args) < 2 {
		help(os.Stdout)
		os.Exit(0)
	}

	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "You must specify a command to run")
		help(os.Stderr)
		os.Exit(1)
	}

	cmd := commands[flag.Args()[0]]
	if cmd == nil {
		fmt.Fprintln(os.Stderr, "unknown command:", flag.Args()[0])
		help(os.Stderr)
		os.Exit(1)
	}

	cmd.f(flag.Args()[1:])
}

func help(w io.Writer) {
	fmt.Fprintln(w, "usage: md2html [command] [command-arguments]")
	fmt.Fprint(w, "\nThe commands are:\n\n")
	for name := range commands {
		fmt.Fprintln(w, "\t", name)
	}
	fmt.Fprintln(w)
}

func serve(args []string) {
	addr := "8080"
	if len(args) >= 1 {
		if _, err := strconv.Atoi(args[0]); err != nil {
			fmt.Fprint(os.Stderr, "You must specify a numeric port for serving content\n\n")
			fmt.Fprintln(os.Stderr, "usage: md2html serve PORT")
			fmt.Fprintln(os.Stderr)
			os.Exit(1)
		}
		addr = args[0]
	}

	addr = ":" + addr

	fmt.Printf("serving at: http://localhost%s\n", addr)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := "." + r.URL.Path

		paths := []string{
			path,
			path + ".md",
			path + ".partial.html",
			strings.TrimSuffix(path, "/") + "/index.html",
			strings.TrimSuffix(path, "/") + "/index.partial.html",
		}

		var (
			b   []byte
			err error
		)
		for _, p := range paths {
			b, err = renderFile(p)

			if err == nil {
				break
			}

			if err != nil && !os.IsNotExist(err) && !strings.HasSuffix(err.Error(), "is a directory") {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		if err != nil {
			http.NotFound(w, r)
			return
		}

		http.ServeContent(w, r, path, time.Unix(0, 0), bytes.NewReader(b))
	})
	log.Fatal(http.ListenAndServe(addr, nil))
}

func convert(args []string) {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, "You must specify an destination path for built docs\n\n")
		fmt.Fprintln(os.Stderr, "usage: md2html build DEST_PATH")
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	dest := args[0]

	fmt.Printf("Converting markdown to: %s\n", dest)
	convertErr := filepath.Walk(".", func(p string, f os.FileInfo, err error) error {
		if strings.HasPrefix(path.Base(p), ".") {
			return nil
		}

		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		var (
			destFile = filepath.Join(dest, p)
			output   []byte
		)

		if strings.HasSuffix(f.Name(), ".md") {
			destFile = strings.TrimSuffix(destFile, ".md")
		} else if strings.HasSuffix(f.Name(), ".partial.html") {
			destFile = strings.TrimSuffix(destFile, ".partial.html")
		} else if strings.HasSuffix(f.Name(), ".html") {
			destFile = strings.TrimSuffix(destFile, ".html")
		}

		output, err = renderFile(p)
		if err != nil {
			return err
		}

		// For serving index files in S3, we need the .html extension.
		if f.Name() == "index.html" || f.Name() == "index.partial.html" {
			destFile += ".html"
		}

		err = os.MkdirAll(filepath.Dir(destFile), 0777) // #nosec
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(destFile, output, 0644)
		if err != nil {
			return err
		}

		fmt.Printf("converted: %s\n", p)
		return nil
	})

	if convertErr != nil {
		fmt.Println(convertErr)
		os.Exit(1)
	}
}

func printe(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func renderFile(p string) ([]byte, error) {
	content, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	templateExtensions := make(map[string]bool)
	for _, v := range []string{".md", ".html", ".js", ".css"} {
		templateExtensions[v] = true
	}

	if strings.HasSuffix(p, ".md") {
		content, err = renderMarkdown(p, content)
	} else if strings.HasSuffix(p, ".partial.html") {
		content, err = renderLayout(p, content)
	} else if templateExtensions[filepath.Ext(p)] {
		content, err = renderTemplate(p, []byte{}, content)
	}

	if err != nil {
		return nil, err
	}

	return content, nil
}

func renderTemplate(p string, content []byte, layout []byte) ([]byte, error) {
	pathClass := strings.Replace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(p, "./"), "../"), ".md"), "/", "_", -1)
	substitution := struct {
		Body     string
		Filename string
	}{string(content), pathClass}

	layoutTemplate, err := template.New(p).Parse(string(layout))
	if err != nil {
		return nil, err
	}
	var x bytes.Buffer
	err = layoutTemplate.Execute(&x, substitution)
	if err != nil {
		return nil, err
	}

	return x.Bytes(), nil
}

func renderMarkdown(p string, src []byte) ([]byte, error) {
	src = interpolateCode(src, path.Dir(p))
	src = preprocessLocalLinks(src)
	src = markdown(src)
	src = formatSidenotes(src)

	html, err := renderLayout(p, src)
	if err != nil {
		return nil, err
	}

	return html, nil
}

// Returns the contents of a layout.html file
// starting in the directory of p and ending at the command's
// working directory.
// If no layout.html file is found, a default layout that renders .Body
// is returned.
func renderLayout(p string, content []byte) ([]byte, error) {
	originalPath := p
	layout := []byte("{{.Body}}")

	// Render any variables
	content, err := renderTemplate(originalPath, []byte{}, content)
	if err != nil {
		return nil, err
	}

	for {
		p = path.Dir(p)
		l, err := ioutil.ReadFile(p + "/layout.html")
		if err == nil {
			layout = l
			break
		}
		if !os.IsNotExist(err) {
			return nil, err
		}

		if p == "." {
			break
		}
	}

	return renderTemplate(originalPath, content, layout)
}

func interpolateCode(md []byte, hostPath string) []byte {
	const pat = `$code `
	w := new(bytes.Buffer)
	scanner := bufio.NewScanner(bytes.NewBuffer(md))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, pat) {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				fmt.Fprintln(w, "Error: invalid snippet:", line)
				continue
			}

			snippet := fields[1]
			paths := fields[2:]

			fmt.Fprintln(w, "<div class='snippet-set'>")

			for _, p := range paths {
				if path.IsAbs(p) {
					p = path.Join(os.Getenv("CHAIN"), p)
				} else {
					p = path.Join(hostPath, p)
				}
				writeCode(w, p, snippet)
			}

			writeCodeSelector(w, paths)

			fmt.Fprintln(w, "</div>")

			continue
		}
		fmt.Fprintln(w, line)
	}
	return w.Bytes()
}

func writeCodeSelector(w io.Writer, paths []string) {
	var exts []string
	for _, p := range paths {
		exts = append(exts, extension(p))
	}
	sort.Strings(exts)

	if len(exts) < 2 {
		return
	}

	fmt.Fprintln(w, "<ul>")
	for _, e := range exts {
		fmt.Fprintln(w, "<li><span data-docs-lang='"+e+"'>")
		fmt.Fprintln(w, extToLang[e])
		fmt.Fprintln(w, "</span></li>")
	}
	fmt.Fprintln(w, "</ul>")
}

func writeCode(w io.Writer, path, snippet string) {
	code, err := readSnippet(path, snippet)
	if err != nil {
		code = err.Error()
	} else {
		code = escapeHTML(removeCommonIndent(code))
	}

	ext := extension(path)

	fmt.Fprintln(w, "<pre class='"+ext+"'><code>"+code+"</code></pre>")
}

func readSnippet(path, snippet string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to read source file: %s: %s", path, err)
	}

	src := string(b)

	// If the snippet is unset, return everything, omitting snippet definitions.
	if snippet == "" {
		var res string
		for _, line := range strings.SplitAfter(src, "\n") {
			if strings.Contains(line, "snippet") {
				continue
			}
			res += line
		}
		return res, nil
	}

	if !strings.Contains(src, "snippet "+snippet) {
		return "", fmt.Errorf("Snippet %q is not in %q", snippet, path)
	}

	var (
		res   []string
		found bool
	)

	for _, line := range strings.Split(src, "\n") {
		if !found && strings.Contains(line, "snippet "+snippet) {
			found = true
			continue
		}

		if found && strings.Contains(line, "endsnippet") {
			break
		}

		if found {
			res = append(res, line)
		}
	}

	return strings.Join(res, "\n"), nil
}

func removeCommonIndent(s string) string {
	lines := strings.Split(s, "\n")
	ci := commonIndent(lines)

	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, ci)
	}

	return strings.Join(lines, "\n")
}

// commonIndent returns the longest indentation shared by all non-empty lines
// in the input.
func commonIndent(lines []string) string {
	var (
		indent = regexp.MustCompile(`^[ \t]+`)
		res    string
		resSet bool
	)

	for _, line := range lines {
		// blank lines should not influence the common indentation
		if strings.TrimSpace(line) == "" {
			continue
		}

		if !resSet {
			res = indent.FindString(line)
			resSet = true
			continue
		}

		res = commonPrefix(res, indent.FindString(line))
		if res == "" {
			break
		}
	}

	return res
}

// commonPrefix returns the longest shared prefix between two strings. It
// assumes the characters can be represented by a single byte.
func commonPrefix(a, b string) string {
	var res []byte
	for i := range a {
		if i >= len(b) || a[i] != b[i] {
			break
		}
		res = append(res, a[i])
	}
	return string(res)
}

func markdown(source []byte) []byte {
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_XHTML
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_DASHES
	htmlFlags |= blackfriday.HTML_FOOTNOTE_RETURN_LINKS
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	extensions := 0
	extensions |= blackfriday.EXTENSION_FOOTNOTES
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_AUTOLINK
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	extensions |= blackfriday.EXTENSION_HEADER_IDS
	extensions |= blackfriday.EXTENSION_LAX_HTML_BLOCKS
	extensions |= blackfriday.EXTENSION_AUTO_HEADER_IDS

	return blackfriday.Markdown(source, renderer, extensions)
}

func escapeHTML(src string) string {
	escapes := map[rune]string{
		'"': "&quot;",
		'&': "&amp;",
		'<': "&lt;",
		'>': "&gt;",
	}

	var (
		runes = []rune(src)
		res   []rune
		start int
	)
	for i, r := range runes {
		if e, ok := escapes[r]; ok {
			res = append(res, runes[start:i]...) // add everything since last escaped char
			res = append(res, []rune(e)...)      // add the escaped version of the char
			start = i + 1
		}
	}

	if start < len(runes) {
		res = append(res, runes[start:]...) // add remainder
	}

	return string(res)
}

// Very rude, but compatible with our code rewrite of local .md links to their post-processed URLs.
// The intention is to process only local links to .md files into non-.md files:
// [Foo](foo.md) -> [Foo](foo)
// [Bar](../bar.md) -> [Bar](../bar)
// [Global](https://github.com/.../global.md) -> [Global](https://github.com/.../global.md)   <- .md must be preserved here!
func preprocessLocalLinks(source []byte) []byte {
	// We use a simple rule - local URLs are those without ":" in them.
	var localMdLink = regexp.MustCompile(`(\]\([^:)#]+)\.md([)#])`)

	w := new(bytes.Buffer)
	scanner := bufio.NewScanner(bytes.NewBuffer(source))
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(w, localMdLink.ReplaceAllString(line, "$1$2"))
	}
	return w.Bytes()
}

func formatSidenotes(source []byte) []byte {
	const openTag = `<p>[sidenote]</p>`
	const closeTag = `<p>[/sidenote]</p>`

	w := new(bytes.Buffer)
	scanner := bufio.NewScanner(bytes.NewBuffer(source))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, closeTag) {
			fmt.Fprintln(w, `</div>`)
			continue
		}
		if strings.HasPrefix(line, openTag) {
			fmt.Fprintln(w, `<div class="sidenote">`)
			continue
		}
		fmt.Fprintln(w, line)
	}
	return w.Bytes()
}

func extension(s string) string {
	toks := strings.Split(s, ".")
	if len(toks) < 2 {
		return ""
	}
	return toks[len(toks)-1]
}
