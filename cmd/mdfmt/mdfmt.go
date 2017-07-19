package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

func main() {
	heading := 0   // current heading level
	blank := false // whether there are pending blank lines

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if line == "" {
			blank = true
			continue
		}
		lineHeading := countPrefix(line, "#")
		if lineHeading > 0 {
			if heading > 0 {
				fmt.Print("\n")
				if lineHeading < heading {
					fmt.Print(strings.Repeat("\n", heading-lineHeading))
				}
			}
			fmt.Println(line)
			blank = false
			heading = lineHeading
			continue
		}
		if blank {
			fmt.Print("\n")
			blank = false
		}
		fmt.Println(line)
	}
}

func countPrefix(s, prefix string) int {
	res := 0
	for strings.HasPrefix(s, prefix) {
		res++
		s = s[len(prefix):]
	}
	return res
}
