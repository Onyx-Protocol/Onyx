package etcdname

import "testing"

func TestFollowJSONPointer(t *testing.T) {
	cases := []struct {
		input   interface{}
		pointer []string
		want    string
	}{
		{
			"i am a string",
			nil,
			"i am a string",
		},
		{
			6, // i am not a string
			nil,
			"",
		},
		{
			nil, // i am DEFINITELY not a string
			nil,
			"",
		},
		{
			map[string]interface{}{"firstKey": "one"},
			[]string{"firstKey"},
			"one",
		},
		{
			map[string]interface{}{},
			[]string{"missingKey"},
			"",
		},
		{
			map[string]interface{}{"~firstKey": "i have a tilde"},
			[]string{"~0firstKey"},
			"i have a tilde",
		},
		{
			map[string]interface{}{"first/key": "i have a slash"},
			[]string{"first~1key"},
			"i have a slash",
		},
		{
			[]interface{}{"index0"},
			[]string{"0"},
			"index0",
		},
		{
			[]interface{}{"index0"},
			[]string{"1"}, // out of bounds
			"",
		},
		{
			[]interface{}{"index0"},
			[]string{"-1"}, // out of bounds
			"",
		},
		{
			[]interface{}{"index0"},
			[]string{"notAnIndex"},
			"",
		},
		{
			map[string]interface{}{
				"firstKey": "one",
				"groupOfKeys": []interface{}{
					map[string]interface{}{"key1": "g1"},
					map[string]interface{}{"key2": "g2"},
				},
			},
			[]string{"groupOfKeys", "1", "key2"},
			"g2",
		},
	}

	for _, c := range cases {
		result := followJSONPointer(c.input, c.pointer)

		if result != c.want {
			t.Fatalf("followJSONPointer(%v, %v) = %s, want=%s", c.input, c.pointer, result, c.want)
		}
	}
}

func TestSplitPointer(t *testing.T) {
	cases := []struct {
		original    string
		wantKey     string
		wantPointer string
	}{
		{"key#pointer#stillpointer", "key", "pointer#stillpointer"},
		{"justakey", "justakey", ""},
	}

	for _, c := range cases {
		key, pointer := splitPointer(c.original)
		if key != c.wantKey || pointer != c.wantPointer {
			t.Fatalf("splitPointer(%s) = %s, %s, want = %s, %s", c.original, key, pointer, c.wantKey, c.wantPointer)
		}
	}
}
