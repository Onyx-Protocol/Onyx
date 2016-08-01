package core

import (
	"golang.org/x/net/context"

	"chain/core/txbuilder"
	"chain/crypto/ed25519/hd25519"
	"chain/encoding/json"
	"chain/errors"
)

func (a *api) mockhsmGenKey(ctx context.Context) (result struct{ XPub json.HexBytes }, err error) {
	xpub, err := a.hsm.GenKey(ctx)
	if err != nil {
		return result, err
	}
	result.XPub = xpub.Bytes()
	return result, nil
}

func (a *api) mockhsmDelKey(ctx context.Context, xpubBytes json.HexBytes) error {
	xpub, err := hd25519.XPubFromBytes(xpubBytes)
	if err != nil {
		return err
	}
	return a.hsm.DelKey(ctx, xpub)
}

func (a *api) mockhsmSignTemplates(ctx context.Context, tpls []*txbuilder.Template) []interface{} {
	resp := make([]interface{}, 0, len(tpls))
	for _, tpl := range tpls {
		err := txbuilder.Sign(ctx, tpl, a.mockhsmSignTemplate)
		if err != nil {
			info, _ := errInfo(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tpl)
		}
	}
	return resp
}

func (a *api) mockhsmSignTemplate(ctx context.Context, sigComponent *txbuilder.SigScriptComponent, sig *txbuilder.Signature) ([]byte, error) {
	xpub, err := hd25519.XPubFromString(sig.XPub)
	if err != nil {
		return nil, errors.Wrap(err, "parsing xpub")
	}
	return a.hsm.Sign(ctx, xpub, sig.DerivationPath, sigComponent.SignatureData[:])
}
