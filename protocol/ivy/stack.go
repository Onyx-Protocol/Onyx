package ivy

type stackEntry string

func (s stackEntry) matches(expr expression) bool {
	return string(s) == expr.String()
}

func addParamsToStack(stack []stackEntry, params []*param) []stackEntry {
	for i := len(params) - 1; i >= 0; i-- {
		p := params[i]
		if p.typ != "Value" {
			stack = append(stack, stackEntry(p.name))
		}
	}
	return stack
}
