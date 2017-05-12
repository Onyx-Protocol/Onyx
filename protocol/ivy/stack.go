package ivy

type stackEntry string

func (s stackEntry) matches(name string) bool {
	return string(s) == name
}

func addParamsToStack(stack []stackEntry, params []*param) []stackEntry {
	for i := len(params) - 1; i >= 0; i-- {
		p := params[i]
		switch p.typ {
		case "Value":
			continue
		case "AssetAmount":
			stack = append(stack, stackEntry(p.name+".amount"))
			stack = append(stack, stackEntry(p.name+".asset"))
		default:
			stack = append(stack, stackEntry(p.name))
		}
	}
	return stack
}
