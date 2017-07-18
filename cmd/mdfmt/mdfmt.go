package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

func main() {
	heading := 0 // current heading level
	blank := 0   // number of pending blank lines

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if line == "" {
			blank++
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
			blank = 0
			heading = lineHeading
			continue
		}
		if blank > 0 {
			fmt.Print("\n")
			blank = 0
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
