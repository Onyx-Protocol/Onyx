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
			`{"program":"547a6416000000000054795679515679c163240000007878ae7bac690000c3c2515779c1","clause_info":[{"name":"trade","value_info":[{"name":"payment","program":"sellerControlProgram","asset_amount":"requested"},{"name":"offered"}]},{"name":"cancel","args":[{"name":"sellerSig","type":"Signature"}],"value_info":[{"name":"offered","program":"sellerControlProgram"}]}]}`,
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
contract TradeOffer(requested: AssetAmount, sellerControlProgram: Program, sellerKey: PublicKey, offered: Value) {
  clause trade(payment: Value) {
    verify payment.assetAmount == requested
    output sellerControlProgram(payment)
    return offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    output sellerControlProgram(offered)
  }
}
`
