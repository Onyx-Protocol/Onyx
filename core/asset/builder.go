package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/encoding/json"
	"chain/errors"
)

type IssueAction struct {
	Params struct {
		bc.AssetAmount
		TTL time.Duration
	}
	ReferenceData json.HexBytes `json:"reference_data"`
}

func (a *IssueAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
	asset, err := Find(ctx, a.Params.AssetID)
	if err != nil {
		return nil, nil, nil, errors.WithDetailf(err, "find asset with ID %q", a.Params.AssetID)
	}
	ttl := a.Params.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	txin := bc.NewIssuanceInput(time.Now(), time.Now().Add(ttl), asset.GenesisHash, a.Params.Amount, asset.IssuanceProgram, a.ReferenceData, nil)
	tplIn := issuanceInput(asset, a.Params.AssetAmount)

	return []*bc.TxInput{txin}, nil, []*txbuilder.Input{tplIn}, nil
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *Asset, aa bc.AssetAmount) *txbuilder.Input {
	tmplInp := &txbuilder.Input{AssetAmount: aa}
	path := signers.Path(a.Signer, signers.AssetKeySpace, a.KeyIndex) // is this the right key index?
	sigs := txbuilder.InputSigs(a.Signer.XPubs, path)
	tmplInp.AddWitnessSigs(sigs, txscript.SigsRequired(a.IssuanceProgram), nil)
	return tmplInp
}
