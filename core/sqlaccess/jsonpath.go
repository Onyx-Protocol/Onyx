package sqlaccess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

func queryPath(jsonValue interface{}, path ...string) (interface{}, error) {
	for i, p := range path {
		if jsonValue == nil {
			return nil, errWrongType{val: jsonValue, path: path[:i]}
		}
		switch v := jsonValue.(type) {
		case []interface{}:
			index, err := strconv.Atoi(p)
			if err != nil {
				return nil, errWrongType{val: jsonValue, path: path[:i]}
			}
			jsonValue = v[index]
		case map[string]interface{}:
			jsonValue = v[p]
		default:
			return nil, errWrongType{val: jsonValue, path: path[:i]}
		}
	}
	return jsonValue, nil
}

type errWrongType struct {
	path []string
	val  interface{}
}

func (err errWrongType) Error() string {
	typ := jsonType(err.val)
	return fmt.Sprintf("unexpected %s at %s", typ, pathString(err.path))
}

func pathString(path []string) string {
	if len(path) == 0 {
		return "root element"
	}
	var buf bytes.Buffer
	for i, p := range path {
		if i > 0 {
			buf.WriteRune('.')
		}
		buf.WriteString(fmt.Sprintf("%q", p))
	}
	return buf.String()
}

func jsonType(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, json.Number:
		return "number"
	default:
		return "<invalid type>"
	}
}
