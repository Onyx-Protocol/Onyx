package main

import (
	"bytes"
	"os"
	"text/template"
)

// Paths must begin with generated/; see commitRevIDs.
var revIDLang = map[string]*template.Template{
	"generated/rev/RevId.java": template.Must(template.New("").Parse(revIDJava)),
	"generated/rev/revid.go":   template.Must(template.New("").Parse(revIDGo)),
	"generated/rev/revid.js":   template.Must(template.New("").Parse(revIDJavaScript)),
	"generated/rev/revid.rb":   template.Must(template.New("").Parse(revIDRuby)),
}

const revIDGo = `package rev

const ID string = "{{.}}"
`

const revIDJava = `
public final class RevId {
	public final String Id = "{{.}}";
}
`

const revIDJavaScript = `
export const rev_id = "{{.}}"
`

const revIDRuby = `
module Chain::Rev
	ID = "{{.}}".freeze
end
`

// revID returns a string to use
// for the revid of the next commit on main.
func revID(landdir, baseBranch string) (string, error) {
	cmd := dirCmd(landdir, "git", "rev-list", "--count", "origin/"+baseBranch, "--")
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return baseBranch + "/rev" + string(bytes.TrimSpace(b)), nil
}
