package name

import (
	"fmt"
	"strings"
)

var escapePairs = []string{
	`\\`, `\`,
	`\"`, `"`,
	`\+`, `+`,
	`\,`, `,`,
	`\;`, `;`,
	`\<`, `<`,
	`\>`, `>`,
	`\ `, ` `,
	`\#`, `#`,
	`\=`, `=`,
	// hex entries added in init
}

var unescaper *strings.Replacer // initialized in init

func init() {
	for b := 0; b <= 255; b++ {
		escapePairs = append(escapePairs,
			fmt.Sprintf(`\%02x`, b),
			string([]byte{byte(b)}),
		)
	}
	unescaper = strings.NewReplacer(escapePairs...)
}
