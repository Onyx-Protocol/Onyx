package ivy

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name     string
		contract string
		wantJSON string
	}{
		{
			"TradeOffer",
			tradeOffer,
			`{"name":"TradeOffer","program":"547a6416000000000052795479515779c1632600000054795479ae7cac690000c3c2515779c1","params":[{"name":"requested","type":"AssetAmount"},{"name":"sellerAddress","type":"Address"},{"name":"sellerKey","type":"PublicKey"},{"name":"offered","type":"Value"}],"clause_info":[{"name":"trade","args":[],"value_info":[{"name":"payment","program":"sellerAddress","asset_amount":"requested"},{"name":"offered"}]},{"name":"cancel","args":[{"name":"sellerSig","type":"Signature"}],"value_info":[{"name":"offered","program":"sellerAddress"}]}]}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.contract)
			got, err := Compile(r)
			if err != nil {
				t.Fatal(err)
			}
			var want CompileResult
			err = json.Unmarshal([]byte(c.wantJSON), &want)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				gotJSON, _ := json.Marshal(got)
				t.Errorf("got %s, want %s", string(gotJSON), c.wantJSON)
			}
		})
	}
}

const tradeOffer = `
contract TradeOffer(requested: AssetAmount, sellerAddress: Address, sellerKey: PublicKey, offered: Value) {
  clause trade(payment: Value) {
    verify payment.assetAmount == requested
    output sellerAddress(payment)
    return offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    output sellerAddress(offered)
  }
}
`
