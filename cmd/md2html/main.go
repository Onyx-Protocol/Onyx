package main

import (
	"bufio"
	"bytes"
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
	"strings"

	"github.com/russross/blackfriday"
)

var layoutPlaceholder = []byte("{{Body}}")
var documentNamePlaceholder = []byte("{{Filename}}")

var extToLang = map[string]string{
	"java": "Java",
	"rb":   "Ruby",
	"js":   "JavaScript",
}

func main() {
	var dest = ":8080"

	if len(os.Args) > 2 {
		log.Fatal("usage: md2html [dest]")
	}
	if len(os.Args) == 2 {
		dest = os.Args[1]
	}

	if !strings.Contains(dest, ":") {
		printe(convert(dest))
		os.Exit(0)
	}

	serve(dest)
}

func serve(addr string) {
	fmt.Printf("serving at: http://localhost%s\n", addr)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := "." + r.URL.Path
		if filepath.Ext(path) != "" {
			http.ServeFile(w, r, path)
			return
		}

		paths := []string{
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
			if strings.HasSuffix(p, ".md") {
				b, err = renderMarkdown(p)
			} else if strings.HasSuffix(p, ".partial.html") {
				b, err = renderHTML(p)
			} else {
				b, err = ioutil.ReadFile(p)
			}

			if err == nil {
				break
			}

			if err != nil && !os.IsNotExist(err) {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Write(b)
	})
	log.Fatal(http.ListenAndServe(addr, nil))
}

func convert(dest string) error {
	fmt.Printf("Converting markdown to: %s\n", dest)
	return filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		var (
			destFile = filepath.Join(dest, path)
			output   []byte
		)

		if strings.HasSuffix(f.Name(), ".md") {
			destFile = strings.TrimSuffix(destFile, ".md")
			output, err = renderMarkdown(path)
		} else if strings.HasSuffix(f.Name(), ".partial.html") {
			destFile = strings.TrimSuffix(destFile, ".partial.html")
			output, err = renderHTML(path)
		} else if strings.HasSuffix(f.Name(), ".html") {
			destFile = strings.TrimSuffix(destFile, ".html")
			output, err = ioutil.ReadFile(path)
		} else {
			output, err = ioutil.ReadFile(path)
		}

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

		fmt.Printf("converted: %s\n", path)
		return nil
	})
}

func renderHTML(path string) ([]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	html, err := layout(path)
	if err != nil {
		return nil, err
	}

	html = bytes.Replace(html, layoutPlaceholder, content, 1)
	return html, nil
}

func printe(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func renderMarkdown(f string) ([]byte, error) {
	src, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	src = interpolateCode(src, path.Dir(f))

	html, err := layout(f)
	if err != nil {
		return nil, err
	}

	src = preprocessLocalLinks(src)
	src = markdown(src)
	src = formatSidenotes(src)

	html = bytes.Replace(html, layoutPlaceholder, src, 1)
	pathClass := strings.Replace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(f, "./"), "../"), ".md"), "/", "_", -1)
	html = bytes.Replace(html, documentNamePlaceholder, []byte(pathClass), -1)

	return html, nil
}

// Returns the contents of a layout.html file
// starting in the directory of p and ending at the command's
// working directory.
// If no layout.html file is found layoutPlaceholder is returned
// as a default layout.
func layout(p string) ([]byte, error) {
	// Don't search for layouts beyond the working dir
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		p = path.Dir(p)
		l, err := ioutil.ReadFile(p + "/layout.html")
		if err == nil {
			return l, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}

		if wd == p {
			break
		}
	}

	return layoutPlaceholder, nil
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
		code = removeCommonIndent(code)
	}

	ext := extension(path)

	fmt.Fprintln(w, "<pre class='"+ext+"'><code>"+code+"</code></pre>")
}

func readSnippet(path, snippet string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("Unable to read source file: %s\n", path)
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
