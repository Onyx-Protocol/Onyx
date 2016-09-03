package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/russross/blackfriday"
)

var (
	layout = []byte("{{Body}}")
	footer []byte
	header []byte
)

func main() {
	var err error
	if len(os.Args) != 3 {
		log.Fatal("usage: md2html srcDir destDir")
	}
	srcDir := os.Args[1]
	destDir := os.Args[2]

	if _, err := os.Stat(srcDir + "/layout.html"); !os.IsNotExist(err) {
		layout, err = ioutil.ReadFile(srcDir + "/layout.html")
		check(err)
	}

	files, err := ioutil.ReadDir(srcDir)
	check(err)

	for i := range files {
		srcName := files[i].Name()
		if !strings.HasSuffix(srcName, ".md") {
			continue
		}
		srcBytes, err := ioutil.ReadFile(srcDir + "/" + srcName)
		check(err)

		destBytes := wrap(convert(append(append(header, srcBytes...), footer...)))
		destName := destDir + "/" + srcName[0:len(srcName)-3]
		err = ioutil.WriteFile(destName, destBytes, 0644)
		check(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func wrap(body []byte) []byte {
	return bytes.Replace(layout, []byte("{{Body}}"), body, 1)
}

func convert(source []byte) []byte {
	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_XHTML
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_DASHES
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	extensions := 0
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
