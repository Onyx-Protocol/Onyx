package migrate

import (
	"regexp"
	"testing"
)

var validName = regexp.MustCompile(`^201[5-9]-[0-9][0-9]-[0-9][0-9].+([0-9]).+([a-z]).+([a-z0-9-])$`)

func TestNames(t *testing.T) {
	list := make([]migration, len(migrations))
	copy(list, migrations)

	for i, m := range list {
		if !validName.MatchString(m.Name) {
			t.Errorf("bad name: %s", m.Name)
		}
		if i > 0 && list[i-1].Name >= m.Name {
			t.Errorf("out of order")
		}
	}

	// Fail if we have more than one of any index
	// on the same day. YYYY-MM-DD.N
	a := make([]string, len(list))
	for i, m := range list {
		a[i] = m.Name[:12]
		if i > 0 && a[i-1] == a[i] {
			t.Errorf("duplicate indexes %s %s", list[i-1].Name, m.Name)
		}
	}
}
