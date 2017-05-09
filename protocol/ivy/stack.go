package ivy

type stackEntry struct {
	param    *param
	property string
}

func (s stackEntry) matches(expr expression) bool {
	if s.param != referencedParam(expr) {
		return false
	}
	if p, ok := expr.(*propRef); ok {
		return s.property == p.property
	}
	return s.property == ""
}

func addParamsToStack(stack []stackEntry, params []*param) []stackEntry {
	for i := len(params) - 1; i >= 0; i-- {
		p := params[i]
		switch p.typ {
		case "Value":
			continue
		case "AssetAmount":
			stack = append(stack, stackEntry{
				param:    p,
				property: "amount",
			})
			stack = append(stack, stackEntry{
				param:    p,
				property: "asset",
			})
		default:
			stack = append(stack, stackEntry{
				param: p,
			})
		}
	}
	return stack
}
