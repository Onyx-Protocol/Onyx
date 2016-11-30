package txbuilder

import (
	"context"

	"chain/core/pb"
	"chain/protocol/bc"
)

type Template struct {
	*pb.TxTemplate
	Tx        *bc.TxData
	sigHasher *bc.SigHasher
}

func (t *Template) Hash(idx uint32) bc.Hash {
	if t.sigHasher == nil {
		t.sigHasher = bc.NewSigHasher(t.Tx)
	}
	return t.sigHasher.Hash(idx)
}

type Action interface {
	Build(context.Context, *TemplateBuilder) error
}
