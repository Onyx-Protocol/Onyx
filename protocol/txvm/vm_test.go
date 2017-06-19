package txvm

import (
	"encoding/hex"
	"testing"
)

func TestVM(t *testing.T) {
	prog, _ := hex.DecodeString("65b8cedafbc82b2665f89dfffbc82b268c0108def52eab3a84316975c2207c8ae006497a570c3349529d047b91962a3e94c3fea57e4a0e22cf0897df8bc38753273c7f5213e2914372f117deaf51cd4285c1adf3f603d013069b698eadf612f06935658a01766baa207595bc9aa6a7a2d29c79df9b4d9057b773ed49b665fdf14232542c1c782807315151ad696c00c07f00000000000000000000000000000000000000000000000000000000000000007fa7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a6064267f7c8ae006497a570c3349529d047b91962a3e94c3fea57e4a0e22cf0897df8bc33d513e6064267f7c8ae006497a570c3349529d047b91962a3e94c3fea57e4a0e22cf0897df8bc33f7fa7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a61beef345f9f01c47a63518201cf3a19a0f497ead35caac403e9828facc0cf19fd3d797b9334e9218cf2b54fc17085948a9de7146b52a0c1f44ae727faaacbf77aef1bbe5ca2078201204c023563a3e9b8dee69fd51385b9d87be9596fdcb0dd99e777e085226dff3475ae8753273535")

	tx := &Tx{
		Proof: prog,
	}
	ok := Validate(tx, TraceOp(func(s stack, op byte, data, p []byte) {
		if op < BaseData {
			t.Logf("%s\t\t#stack: %d", OpNames[op], s.Len())
		} else {
			t.Logf("[%x]\t\t#stack: %d", data, s.Len())
		}
	}), TraceError(func(err error) { t.Log(err) }))
	if !ok {
		t.Error("expected ok")
	}
}
