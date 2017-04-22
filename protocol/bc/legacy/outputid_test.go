package legacy

import (
	"chain/protocol/bc"
	"testing"
)

func TestRetirementOutputID(t *testing.T) {
	const (
		hexA = `07010700c4daac98b92b0001019201018f0122368c0dd81bf595846fb9ccfe3be2d7a2cb6e97f8b4967ba454349bd7f82afe8227a59c46f5ab97630654881b5c178d21c93d95c931e07c843da1d221a8400f6400012b766baa20fd097c6edeb629551879d00720dea58ccd6379af3d793675a1a9131ad656aa045151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a0067030040899c4febe576b475cc633c409469064ef6a1ec1ec0a802cf5de6cfca94af95c2d3d9617d2891de9a3767beab48c7e5356fbb3936bbbbb7c7373d9e64d8bf0e032320c586ad8e743d0eb83fc60d49e550f061eaee8dfa3d8c882ff58eeb06b28eaae5ae8702014e8227a59c46f5ab97630654881b5c178d21c93d95c931e07c843da1d221a8400f5f012b766baa2036c0d99b7950605ad9c49f0811b9622d07140920561b28bac30648ec55aae27a5151ad696c00c0000001248227a59c46f5ab97630654881b5c178d21c93d95c931e07c843da1d221a8400f0501016a000000`
		hexB = `07010700e7b59998b92b0001019201018f0122368c0dd81bf595846fb9ccfe3be2d7a2cb6e97f8b4967ba454349bd7f82afe8227a59c46f5ab97630654881b5c178d21c93d95c931e07c843da1d221a8400f6400012b766baa20fd097c6edeb629551879d00720dea58ccd6379af3d793675a1a9131ad656aa045151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a0067030040fffbdaa823e911069c8a33754204bd744faf401691cc198e6cc9784f4b77c6ca732c7f6b4393bb910edd71925fe16746ed1f6bc7d7a97241057d1f12ab21670123204221068c8b12e73d0b61269d879e45e618ed1de78c6958ccbb15eaee4f96a795ae8702014e8227a59c46f5ab97630654881b5c178d21c93d95c931e07c843da1d221a8400f5f012b766baa20f92f105f3c197e3e4c2d39bc03e029338806d7767ff4b8eb948b3093ebeb0de35151ad696c00c0000001248227a59c46f5ab97630654881b5c178d21c93d95c931e07c843da1d221a8400f0501016a000000`
	)

	var a, b Tx
	var err error
	err = a.UnmarshalText([]byte(hexA))
	if err != nil {
		t.Fatal(err)
	}
	err = b.UnmarshalText([]byte(hexB))
	if err != nil {
		t.Fatal(err)
	}
	h1, h2 := a.OutputID(1), b.OutputID(1)
	a_prime, _ := MapTx(&a.TxData)
	for _, e := range a_prime.Entries {
		switch v := e.(type) {
		case *bc.Retirement:
			t.Logf("Source of a_prime: %#v\n", v.Source.Ref.String())
		}
	}
	b_prime, _ := MapTx(&b.TxData)
	for _, e := range b_prime.Entries {
		switch v := e.(type) {
		case *bc.Retirement:
			t.Logf("Source of b_prime: %#v\n", v.Source.Ref.String())
		}
	}
	if *h1 == *h2 {
		t.Logf("a = %#v\n", a.Outputs[1])
		t.Logf("b = %#v\n", b.Outputs[1])
		t.Errorf("retirement outputs have same output ID: %s", h1.String())
	}
}
