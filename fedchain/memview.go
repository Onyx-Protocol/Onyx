package fedchain

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
)

type MemView struct {
	Outs               map[bc.Outpoint]*state.Output
	outsByContractHash map[bc.ContractHash][]*state.Output
	ADPs               map[bc.AssetID]*bc.AssetDefinitionPointer
}

func NewMemView() *MemView {
	return &MemView{
		Outs:               make(map[bc.Outpoint]*state.Output),
		outsByContractHash: make(map[bc.ContractHash][]*state.Output),
		ADPs:               make(map[bc.AssetID]*bc.AssetDefinitionPointer),
	}
}

func (v *MemView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	return v.Outs[p]
}

func (v *MemView) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	return v.ADPs[assetID]
}

func (v *MemView) SaveOutput(o *state.Output) {
	v.Outs[o.Outpoint] = o

	isPayToContract, contractHash, _ := txscript.TestPayToContract(o.TxOutput.Script)
	if isPayToContract {
		v.outsByContractHash[*contractHash] = append(v.outsByContractHash[*contractHash], o)
	}
}

func (v *MemView) SaveAssetDefinitionPointer(adp *bc.AssetDefinitionPointer) {
	v.ADPs[adp.AssetID] = adp
}
