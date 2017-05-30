package compiler

import "fmt"

// name-binding environment
type environ struct {
	entries map[string]*envEntry
	parent  *environ
}

type envEntry struct {
	t typeDesc
	r role
	c *Contract // if t == contractType
}

type role int

const (
	roleKeyword role = 1 + iota
	roleBuiltin
	roleContract
	roleContractParam
	roleContractValue
	roleClause
	roleClauseParam
	roleClauseValue
)

var roleDesc = map[role]string{
	roleKeyword:       "keyword",
	roleBuiltin:       "built-in function",
	roleContract:      "contract",
	roleContractParam: "contract parameter",
	roleContractValue: "contract value",
	roleClause:        "clause",
	roleClauseParam:   "clause parameter",
	roleClauseValue:   "clause value",
}

func newEnviron(parent *environ) *environ {
	return &environ{
		entries: make(map[string]*envEntry),
		parent:  parent,
	}
}

func (e *environ) add(name string, t typeDesc, r role) error {
	if entry := e.lookup(name); entry != nil {
		return fmt.Errorf("%s \"%s\" conflicts with %s", roleDesc[r], name, roleDesc[entry.r])
	}
	e.entries[name] = &envEntry{t: t, r: r}
	return nil
}

func (e *environ) addContract(contract *Contract) error {
	if entry := e.lookup(contract.Name); entry != nil {
		return fmt.Errorf("%s \"%s\" conflicts with %s", roleDesc[roleContract], contract.Name, roleDesc[entry.r])
	}
	e.entries[contract.Name] = &envEntry{t: contractType, r: roleContract, c: contract}
	return nil
}

func (e environ) lookup(name string) *envEntry {
	if res, ok := e.entries[name]; ok {
		return res
	}
	if e.parent != nil {
		return e.parent.lookup(name)
	}
	return nil
}
