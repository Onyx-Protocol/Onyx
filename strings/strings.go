package strings

// Uniq removes adjacent duplicates from a
// and returns the shortened slice.
// To remove non-adjacent duplicates,
// sort the input slice first.
func Uniq(a []string) []string {
	if len(a) == 0 {
		return a
	}
	j := 0
	for i := 1; i < len(a); i++ {
		if a[i] != a[j] {
			j++
			a[j] = a[i]
		}
	}
	return a[:j+1]
}
