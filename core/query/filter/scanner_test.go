package filter

import (
	"testing"

	"chain/testutil"
)

type scannedTok struct {
	pos int
	lit string
	tok token
}

func TestScannerValid(t *testing.T) {
	testCases := []struct {
		input []byte
		toks  []scannedTok
	}{
		{
			input: []byte{},
			toks:  []scannedTok{{pos: 0, lit: "", tok: tokEOF}},
		},
		{
			input: []byte("hello"),
			toks: []scannedTok{
				{pos: 0, lit: "hello", tok: tokIdent},
				{pos: 5, lit: "", tok: tokEOF},
			},
		},
		{
			input: []byte(" AND "),
			toks: []scannedTok{
				{pos: 1, lit: "AND", tok: tokKeyword},
				{pos: 5, lit: "", tok: tokEOF},
			},
		},
		{
			input: []byte("   '   hello   ' "),
			toks: []scannedTok{
				{pos: 3, lit: "'   hello   '", tok: tokString},
				{pos: 17, lit: "", tok: tokEOF},
			},
		},
		{
			input: []byte("inputs(asset_id = 'abcd')"),
			toks: []scannedTok{
				{pos: 0, lit: "inputs", tok: tokIdent},
				{pos: 6, lit: "(", tok: tokPunct},
				{pos: 7, lit: "asset_id", tok: tokIdent},
				{pos: 16, lit: "=", tok: tokPunct},
				{pos: 18, lit: "'abcd'", tok: tokString},
				{pos: 24, lit: ")", tok: tokPunct},
				{pos: 25, lit: "", tok: tokEOF},
			},
		},
	}

	for _, tc := range testCases {
		var s scanner
		s.init(tc.input)

		var got []scannedTok
		var curr scannedTok
		for curr.tok != tokEOF {
			curr.pos, curr.tok, curr.lit = s.Scan()
			got = append(got, curr)
		}
		if !testutil.DeepEqual(got, tc.toks) {
			t.Errorf("Scanning %s got\n%#v\nwant\n%#v\n", tc.input, got, tc.toks)
		}
	}
}

func TestScannerInvalid(t *testing.T) {
	testCases := []struct {
		input []byte
		err   error
	}{
		{
			input: []byte(`hello\`),
			err:   parseError{pos: 5, msg: `illegal character '\\'`},
		},
		{
			input: []byte(`'hello\''`),
			err:   parseError{pos: 0, msg: `illegal backslash in string literal`},
		},
		{
			input: []byte(`'hello\'`),
			err:   parseError{pos: 0, msg: `illegal backslash in string literal`},
		},
		{
			input: []byte(`'hello`),
			err:   parseError{pos: 0, msg: `string literal not terminated`},
		},
		{
			input: append([]byte(`hello`), 0),
			err:   parseError{pos: 5, msg: `illegal character NUL`},
		},
		{
			input: []byte(`comme ci comme ça`),
			err:   parseError{pos: 15, msg: `non-ASCII character`},
		},
		{
			input: []byte(`'comme ci comme ça'`),
			err:   parseError{pos: 16, msg: `non-ASCII character`},
		},
		{
			input: []byte(`asset_definition.fund_manager.résumé`),
			err:   parseError{pos: 31, msg: `non-ASCII character`},
		},
		{
			input: []byte(`asset_alias = '区块链'`),
			err:   parseError{pos: 15, msg: `non-ASCII character`},
		},
		{
			input: []byte(`0xwhat`),
			err:   parseError{pos: 0, msg: `illegal hexadecimal number`},
		},
		{
			input: []byte(`10 = 02`),
			err:   parseError{pos: 5, msg: `illegal leading 0 in number`},
		},
	}

	for _, tc := range testCases {
		err := func([]byte) (err error) {
			defer func() {
				if e, ok := recover().(error); ok {
					err = e
				}
			}()

			var s scanner
			s.init(tc.input)

			var curr scannedTok
			for curr.tok != tokEOF {
				curr.pos, curr.tok, curr.lit = s.Scan()
			}
			return err
		}(tc.input)

		if !testutil.DeepEqual(err, tc.err) {
			t.Errorf("Scanning %s got error %s, want error %s", tc.input, err, tc.err)
		}
	}
}
