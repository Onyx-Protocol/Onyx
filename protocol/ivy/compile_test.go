package ivy

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name     string
		contract string
		wantHex  string
	}{
		{
			"TradeOffer",
			tradeOffer,
			"557a6416000000000056795879515879c1632400000076aa537987690000c3c2515879c1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.contract)
			got, err := Compile(r)
			if err != nil {
				t.Fatal(err)
			}
			want, err := hex.DecodeString(c.wantHex)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("got %x, want %x", got, want)
			}
		})
	}
}

const tradeOffer = `
contract TradeOffer(requested: AssetAmount, seller: Program, cancelHash: Hash, offered: Value) {
  clause Trade(payment: Value) {
    verify payment.assetAmount == requested
    output seller(payment)
    return offered
  }
  clause Cancel(cancelSecret: String) {
    verify sha3(cancelSecret) == cancelHash
    output seller(offered)
  }
}
`
