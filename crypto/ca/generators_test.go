package ca

import (
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestZeroPoint(t *testing.T) {

	want := fromHex("0100000000000000000000000000000000000000000000000000000000000000")
	got := ecmath.ZeroPoint.Encode()

	if !constTimeEqual(want, got[:]) {
		t.Errorf("ZeroPoint is encoded as %x, but must be %x", got, want)
	}
}

func TestGeneratorG(t *testing.T) {

	want := fromHex("5866666666666666666666666666666666666666666666666666666666666666")
	got := G.Encode()

	if !constTimeEqual(want, got[:]) {
		t.Errorf("G is encoded as %x, but must be %x", got, want)
	}
}

func TestGeneratorJ(t *testing.T) {
	want := fromHex("00c774b875ed4e395ebb0782b4d93db838d3c4c0840bc970570517555ca71b77")
	got := J.Encode()

	if !constTimeEqual(want, got[:]) {
		t.Errorf("J is encoded as %x, but must be %x", got, want)
	}
}

func TestGeneratorsGi(t *testing.T) {
	wants := [][]string{
		[]string{"00", "e68528ab16b201331fc980c33eef08f7d114554715d370a2c614182ef296dab3"},
		[]string{"03", "32011e4f5c29bbc20d5c96500e87e2303a004687895b2d6d944ff687d0dbefad"},
		[]string{"00", "0d688b311df06d633ced925c1561bea9608f305781c1ab32c55944628181cd1e"},
		[]string{"01", "e17522742ed8bd11aa5d1f2e341400eb1c6f85b47c46817ea0e90b5d5510b420"},
		[]string{"00", "67454d0f02d3962508b89d4209996943825dbf261e7e6e07a842d45b33b2baad"},
		[]string{"02", "c7f0c5eebcb5f37194b7ab96af66e79e0aa37a6cdbde5fbd6af13637b6f05cab"},
		[]string{"02", "c572d7c6f3ef692efbd13928dad208c4572ffe371d88f70a763af3a11cac8709"},
		[]string{"01", "e450cc93f07e0c18a79c1f0572a6971da37bfa81c6003835acf68a8afc1ca33b"},
		[]string{"05", "409ae3e34c0ff3929bceaf7b934923809b461038a1d31c7a0928c8c7ab707604"},
		[]string{"03", "c43d0400219b6745b95ff81176dfbbd5d33b9cc869e171411fff96656273b96c"},
		[]string{"04", "d1eeee54b75cc277bf8a6454accce6086ab24750b0d58a11fb7cad35eba42ff6"},
		[]string{"01", "2446b2efa69fb26a4268037909c466c9b5083bfecf3c2ab3a420114a6f91f0eb"},
		[]string{"00", "d0c4ee744ac129d0282a1554ca7a339e3d9db740826d365eefe984c0e5023969"},
		[]string{"00", "e1d621717a876830e0c7c1bf8e7e674cf5cbe3aa1e71885d7d347854277aa6ca"},
		[]string{"02", "6e95425b9481a70aa553f1e7d30b8182ef659f94ec4e446bc61058615460cbcc"},
		[]string{"03", "4200e80a3976d66f392c7998aa889e6a9efdc65abb6d39575ee8fd8b295008ad"},
		[]string{"06", "3e3e626d2c051c82de750847ced96e1f6af5f4a706703512914c0e334c3cf76e"},
		[]string{"01", "b98d0b73da8ae83754bc61c550c2c6ad76f78ba66e601c3876aea38e086552ae"},
		[]string{"00", "90128059cb3b5baa3b1230e2ef211257253d477490e182bcb60c89bae43752fe"},
		[]string{"00", "b04be209278413859ad51cf6d4a7f15bc2dea9f71c34f71945469705c3885b27"},
		[]string{"01", "fda85012a00938e6f12f4da3cb1642cd1963295d3b089dcb0ee81e73e1b14050"},
		[]string{"00", "73f1392e664fa1687983fcb1c7397b89876f6da8357ee8b07cb44534bc160644"},
		[]string{"00", "0f347deffff466dec1af40197d39e97933112af29d6f305734dc7a4c6e2aceaf"},
		[]string{"00", "c9c779f2644195546a17991a455a6d16a446305f80605e8466f5cd0861a6cb48"},
		[]string{"04", "56614c7cbd1f4b27100d84bd76b4e472237e09ad0970745da252ef0b197291b1"},
		[]string{"00", "4b266eaac77da3229fd884b4fc8163d8fae10a914334805a80b93da1ea8cb7ab"},
		[]string{"00", "e1b33961996a81b591fd54b72b67fe23c3bfac82223713865a39e9802c8a393e"},
		[]string{"01", "f1a19594ea8a6caa753c03d3e63a545ad8dc5ee331647bfeb7a9ac5b21cc04d8"},
		[]string{"00", "60f79007f42376ed140fe7efd43218106613546d8cb3bd06a5cef2e73b02fad7"},
		[]string{"02", "e9cb7b6fd3bb865dac6cff479bc2e3ce98ab95e4a6a57d81ae6d6cb032375f4a"},
		[]string{"01", "7ee2183153687344e093278bc692c4915761ada87a51a778b605e88078d9902a"},
	}
	for i := 0; i < len(wants); i++ {
		cntbuf := fromHex(wants[i][0])
		want := fromHex(wants[i][1])

		p, cnt := makeGi(byte(i))
		if !p.ConstTimeEqual(&Gi[i]) {
			pbytes := p.Encode()
			prepbytes := Gi[i].Encode()
			t.Errorf("Precomputed G%d is %x, but must be %x", i, prepbytes, pbytes)
		}
		if uint64(cntbuf[0]) != cnt {
			t.Errorf("G%d has counter %d, but must use %d", i, cnt, cntbuf[0])
		}
		got := p.Encode()
		if !constTimeEqual(want, got[:]) {
			t.Errorf("G%d is encoded as %x, but must be %x", i, got, want)
		}
	}
}
