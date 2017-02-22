package main

import (
	"bytes"
	"os"
	"text/template"
)

// Paths must begin with generated/; see commitRevIDs.
var revIDLang = map[string]*template.Template{
	// tktk are these paths appropriate?
	"generated/rev/RevId.java": template.Must(template.New("").Parse(revIDJava)),
	"generated/rev/revid.go":   template.Must(template.New("").Parse(revIDGo)),
	"generated/rev/revid.js":   template.Must(template.New("").Parse(revIDJavaScript)),
	"generated/rev/revid.rb":   template.Must(template.New("").Parse(revIDRuby)),
}

const revIDGo = `
package rev
var ID string = "{{.}}"
`

// tktk do java things need to go in a special place?
const revIDJava = `
public final class RevId {
	public final String Id = "{{.}}";
}
`

// tktk [i have no idea what i'm doing dot jpeg]
const revIDJavaScript = `
export const RevID = "{{.}}"
`

// tktk rubby idk; please look
const revIDRuby = `
module Chain::Rev
	ID = "{{.}}".freeze
end
`

// revID returns a string to use
// for the revid of the next commit on main.
func revID(landdir string) (string, error) {
	cmd := dirCmd(landdir, "git", "rev-list", "--count", "main")
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return "rev" + string(bytes.TrimSpace(b)), nil
}
