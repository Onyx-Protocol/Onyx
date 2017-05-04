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
			`{"program":"547a6416000000000054795679515679c1632400000076aa527987690000c3c2515779c1","clause_info":[{"name":"Trade","value_info":[{"name":"payment","program":"seller","asset_amount":"requested"},{"name":"offered"}]},{"name":"Cancel","args":[{"name":"cancelSecret","type":"String"}],"value_info":[{"name":"offered","program":"seller"}]}]}`,
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
