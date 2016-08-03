package cql

type environment interface {
	Get(name string) value
	Environments(name string) []environment
}

type mapEnv map[string]interface{}

func (m mapEnv) Get(name string) value {
	val, ok := m[name]
	if !ok {
		panic("unknown attribute `" + name + "`")
	}

	switch v := val.(type) {
	case bool:
		return value{t: Bool, set: Set{Invert: v}}
	case int:
		return value{t: Integer, integer: v}
	case string:
		return value{t: String, str: v}
	case []interface{}:
		var strs []string
		for _, v := range v {
			s, ok := v.(string)
			if !ok {
				panic("invalid type for attribute `" + name + "`")
			}
			strs = append(strs, s)
		}
		return value{t: List, list: strs}
	default:
		panic("invalid type for attribute `" + name + "`")
	}
}

func (m mapEnv) Environments(name string) []environment {
	val, ok := m[name]
	if !ok {
		panic("unknown attribute `" + name + "`")
	}
	objs, ok := val.([]interface{})
	if !ok {
		panic("invalid type for sub-environment: `" + name + "`")
	}

	var envs []environment
	for _, o := range objs {
		obj, ok := o.(map[string]interface{})
		if !ok {
			panic("invalid type for sub-environment: `" + name + "`")
		}
		envs = append(envs, mapEnv(obj))
	}
	return envs
}
