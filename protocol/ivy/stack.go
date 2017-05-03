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
	for _, p := range params {
		if p.typ == "AssetAmount" {
			stack = append(stack, stackEntry{
				param:    p,
				property: "asset",
			})
			stack = append(stack, stackEntry{
				param:    p,
				property: "amount",
			})
		} else {
			stack = append(stack, stackEntry{
				param: p,
			})
		}
	}
	return stack
}
