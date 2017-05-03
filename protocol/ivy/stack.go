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

func addParamsToStack(stack []stackEntry, params []*param, isContract bool) []stackEntry {
	for _, p := range params {
		switch p.typ {
		case "Value":
			if isContract {
				continue
			}
			fallthrough
		case "AssetAmount":
			stack = append(stack, stackEntry{
				param:    p,
				property: "asset",
			})
			stack = append(stack, stackEntry{
				param:    p,
				property: "amount",
			})
		default:
			stack = append(stack, stackEntry{
				param: p,
			})
		}
	}
	return stack
}
