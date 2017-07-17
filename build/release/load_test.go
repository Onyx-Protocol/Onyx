package release

import (
	"strings"
	"testing"
)

func TestParseError(t *testing.T) {
	invalid := []string{
		"" +
			"* 1 aaaaaaaaaa n/a X\n", // bad product name
		"" +
			"p x aaaaaaaaaa n/a X\n", // bad version string
		"" +
			"p 1.2.3.4 aaaaaaaaaa n/a X\n", // version string too long
		"" +
			"p 1.1 aaaaaaaaaa n/a X\n" +
			"p 1.1 bbbbbbbbbb n/a X\n", // duplicate entry
		"" +
			"p 1.1 aaaaaaaaaa n/a X\n" +
			"p 1.3 bbbbbbbbbb n/a X\n", // gap in version sequence
		"" +
			"p 1.1 aaaaaaaaaa n/a X\n" +
			"p 1.2rc1 bbbbbbbbbb n/a X\n" +
			"p 1.5 cccccccccc n/a X\n", // gap in version sequence
		"" +
			"p 1.1.1 aaaaaaaaaa n/a X\n" +
			"p 1.2.1 bbbbbbbbbb n/a X\n", // point releases in two major releases
		"" +
			"p 1 aaaaaaaaa n/a X\n", // too short commit id
		"" +
			"p 1 aaaaaaaaaaa n/a X\n", // too long commit id
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
			"p 1.1 aaaaaaaaaa n/a X\n" +
			"p 1.2 bbbbbbbbbb n/a X Y\n",
		"" +
			"p 1rc1 aaaaaaaaaa n/a X\n" +
			"p 1 bbbbbbbbbb n/a X\n", // rc before earliest possible version number
		"" +
			"p 1.1 aaaaaaaaaa n/a X\n" +
			"p 1.2rc1 bbbbbbbbbb n/a X\n" +
			"p 1.2 cccccccccc n/a X\n",
	}

	for _, s := range valid {
		_, err := parse(strings.NewReader(s))
		if err != nil {
			t.Errorf("parse(%q) error = %v, want nil", s, err)
		}
	}
}
