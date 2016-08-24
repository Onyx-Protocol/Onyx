package filter

// Set represents the set of parameter values that can satisfy the
// predicate for the provided object.
//
// The zero value of Set is the empty set.
type Set struct {
	Values []string
	Invert bool
}

func intersection(s1, s2 Set) Set {
	if !s1.Invert && !s2.Invert {
		return Set{Values: stringsIntersection(s1.Values, s2.Values)}
	}
	if s1.Invert && s2.Invert {
		return Set{Invert: true, Values: stringsUnion(s1.Values, s2.Values)}
	}

	// Either s1 or s2 is inverted.
	inclusive, exclusive := s1, s2
	if inclusive.Invert {
		inclusive, exclusive = exclusive, inclusive
	}
	return Set{
		Values: stringsSubtract(inclusive.Values, exclusive.Values),
	}
}

func union(s1, s2 Set) Set {
	if !s1.Invert && !s2.Invert {
		return Set{Values: stringsUnion(s1.Values, s2.Values)}
	}
	if s1.Invert && s2.Invert {
		return Set{
			Invert: true,
			Values: stringsIntersection(s1.Values, s2.Values),
		}
	}

	// Either s1 or s2 is inverted.
	inclusive, exclusive := s1, s2
	if inclusive.Invert {
		inclusive, exclusive = exclusive, inclusive
	}
	return Set{
		Invert: true,
		Values: stringsSubtract(exclusive.Values, inclusive.Values),
	}
}

func complement(s Set) Set {
	return Set{Values: s.Values, Invert: !s.Invert}
}

func stringsUnion(s1, s2 []string) (res []string) {
	m := map[string]bool{}
	for _, s := range s1 {
		m[s] = true
	}
	for _, s := range s2 {
		m[s] = true
	}
	for s := range m {
		res = append(res, s)
	}
	return res
}

func stringsIntersection(s1, s2 []string) (res []string) {
	m := map[string]bool{}
	for _, s := range s1 {
		m[s] = true
	}
	for _, s := range s2 {
		if m[s] {
			res = append(res, s)
		}
	}
	return res
}

// stringsSubtract removes all strings in s2 from s1 and returns the
// resulting string slice.
func stringsSubtract(s1, s2 []string) (res []string) {
	remove := map[string]bool{}
	for _, s := range s2 {
		remove[s] = true
	}
	for _, s := range s1 {
		if !remove[s] {
			res = append(res, s)
		}
	}
	return res
}
