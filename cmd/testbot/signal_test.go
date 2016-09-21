package main

import "testing"

func TestSignalWriter(t *testing.T) {
	cases := [][][]byte{
		{[]byte("foobarbaz")},
		{[]byte("fooba"), []byte("rbaz")},
		{[]byte("f"), []byte("o"), []byte("o"), []byte("b"), []byte("a"), []byte("r")},
	}

	for _, test := range cases {
		ch := make(chan struct{})
		w := &signalWriter{
			target: []byte("foobar"),
			done:   ch,
		}

		for i, b := range test {
			if w.done != ch {
				t.Errorf("after %s: done should be ch", test[:i])
			}
			select {
			case <-ch:
				t.Errorf("after %s: channel should be blocked", test[:i])
			default:
			}
			w.Write(b)
		}

		if w.done != nil {
			t.Errorf("%s: done should be nil", test)
		}
		select {
		case _, open := <-ch:
			if open {
				t.Errorf("%s: channel should be closed", test)
			}
		default:
			t.Errorf("%s: channel should be unblocked", test)
		}
	}
}
