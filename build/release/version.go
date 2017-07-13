package release

import (
	"strconv"
	"strings"
)

// Less returns whether version string a is considered "less than" b.
// Both a and b must be a valid version or the empty string,
// otherwise the return value is undefined
// (and not guaranteed to be transitively consistent).
// The empty string is considered less than all versions.
func Less(a, b string) bool {
	a = strings.TrimLeft(a, ".")
	b = strings.TrimLeft(b, ".")
	if a == "" && b == "" {
		return false // equal
	}
	sa, sb := splitvseg(a), splitvseg(b)
	na, nb := decodevseg(sa), decodevseg(sb)
	if na != nb {
		return na < nb
	}
	return Less(a[len(sa):], b[len(sb):])
}

func splitvseg(s string) string {
	i := strings.IndexByte(s, '.')
	if i == -1 {
		i = len(s)
	}
	return s[:i]
}

// decodevseg decodes segment s into a sortable
// numeric form. A segment is either a decimal
// number like "42" or two decimal numbers
// separated by the string "rc" such as "5rc1".
// All "rc" segments sort before their corresponding
// "non-rc" segments, so 5rc1 < 5rc2 < 5.
// Only numbers up to 65,534 are supported.
func decodevseg(s string) int {
	rc := 1<<16 - 1 // no "rc" number is treated as rc65535
	if i := strings.Index(s, "rc"); i >= 0 {
		rc, _ = strconv.Atoi(s[i+2:])
		s = s[:i]
	}
	n, _ := strconv.Atoi(s)
	return n<<16 | rc
}

// Prev returns the "previous" version string from v,
// or the empty string if there is no previous version
// (such as for version 1).
// If v is not a valid version, the return value is undefined.
// Examples:
//   Prev("2")      == "1"
//   Prev("2.1")    == "2"
//   Prev("2.2")    == "2.1"
//   Prev("2.5rc1") == "2.4"
//   Prev("2.5rc5") == "2.4"
//   Prev("2.5")    == "2.4"
//   Prev("2.5.2")  == "2.5.1"
//   Prev("1")      == ""
// Less(Prev(v), v) returns true for any valid version v.
func Prev(v string) string {
	if i := strings.Index(v, "rc"); i >= 0 {
		v = v[:i]
	}
	segs := strings.Split(v, ".")
	if segs[len(segs)-1] == "1" {
		segs = segs[:len(segs)-1]
	} else {
		n, _ := strconv.Atoi(segs[len(segs)-1])
		segs[len(segs)-1] = strconv.Itoa(n - 1)
	}

	return strings.Join(segs, ".")
}

// Branch returns the git branch name where version v can be found.
// It is X.Y-stable for point releases
// (of the form X.Y.Z or X.Y.ZrcW)
// and main otherwise.
func Branch(v string) string {
	if strings.Count(v, ".") != 2 {
		return "main"
	}
	return v[:strings.LastIndex(v, ".")] + "-stable"
}
