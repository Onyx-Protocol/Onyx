package release

import (
	"strings"
	"testing"
)

func TestParseError(t *testing.T) {
	invalid := []string{
		"" +
			"* 1 aaaaaaaaaa na X\n", // bad product name
		"" +
			"p x aaaaaaaaaa na X\n", // bad version string
		"" +
			"p 1.2.3.4 aaaaaaaaaa na X\n", // version string too long
		"" +
			"p 1.1 aaaaaaaaaa na X\n" +
			"p 1.1 bbbbbbbbbb na X\n", // duplicate entry
		"" +
			"p 1.1 aaaaaaaaaa na X\n" +
			"p 1.3 bbbbbbbbbb na X\n", // gap in version sequence
		"" +
			"p 1.1 aaaaaaaaaa na X\n" +
			"p 1.2rc1 bbbbbbbbbb na X\n" +
			"p 1.5 cccccccccc na X\n", // gap in version sequence
		"" +
			"p 1.1.1 aaaaaaaaaa na X\n" +
			"p 1.2.1 bbbbbbbbbb na X\n", // point releases in two major releases
	}

	for _, s := range invalid {
		_, err := parse(strings.NewReader(s))
		if err == nil {
			t.Errorf("parse(%q) error = nil, want error", s)
		}
	}
}

func TestParse(t *testing.T) {
	valid := []string{
		"" +
			"p 1.1 aaaaaaaaaa na X\n" +
			"p 1.2 bbbbbbbbbb na X Y\n",
		"" +
			"p 1rc1 aaaaaaaaaa na X\n" +
			"p 1 bbbbbbbbbb na X\n", // rc before earliest possible version number
		"" +
			"p 1.1 aaaaaaaaaa na X\n" +
			"p 1.2rc1 bbbbbbbbbb na X\n" +
			"p 1.2 cccccccccc na X\n",
	}

	for _, s := range valid {
		_, err := parse(strings.NewReader(s))
		if err != nil {
			t.Errorf("parse(%q) error = %v, want nil", s, err)
		}
	}
}
