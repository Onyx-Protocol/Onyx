package ivy

type stackEntry string

func (s stackEntry) matches(expr expression) bool {
	return string(s) == expr.String()
}

func addParamsToStack(stack []stackEntry, params []*Param, reversed bool) []stackEntry {
	if reversed {
		for i := len(params) - 1; i >= 0; i-- {
			p := params[i]
			if p.Type != "Value" {
				stack = append(stack, stackEntry(p.Name))
			}
		}
	} else {
		for _, p := range params {
			if p.Type != "Value" {
				stack = append(stack, stackEntry(p.Name))
			}
		}
	}
	return stack
}
