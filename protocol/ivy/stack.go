package ivy

type stackEntry struct {
	param    *param
	property string
}

func (e stackEntry) matches(r *ref) bool {
	if e.param == nil {
		return false
	}
	if e.param.name != r.names[0] {
		return false
	}
	if e.property == "" {
		return len(r.names) == 1
	}
	if len(r.names) != 2 {
		return false
	}
	return e.property == r.names[1]
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
