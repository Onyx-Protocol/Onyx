package asset

import (
	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
)

type outputList []*txdb.Output
type OutsByContractHash map[bc.ContractHash]outputList

type MemView struct {
	Outs               map[bc.Outpoint]*txdb.Output
	outsByContractHash OutsByContractHash
	ADPs               map[bc.AssetID]*bc.AssetDefinitionPointer
}

var _ state.View = (*MemView)(nil)

func NewMemView() *MemView {
	return &MemView{
		Outs:               make(map[bc.Outpoint]*txdb.Output),
		outsByContractHash: make(OutsByContractHash),
		ADPs:               make(map[bc.AssetID]*bc.AssetDefinitionPointer),
	}
}

func (v *MemView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	o := v.Outs[p]
	if o == nil {
		return nil
	}
	return &o.Output
}

func (v *MemView) UnspentP2COutputs(ctx context.Context, contractHash bc.ContractHash, assetID bc.AssetID) (result []*state.Output) {
	if outputs, ok := v.outsByContractHash[contractHash]; ok {
		for _, output := range outputs {
			if !output.Output.Spent && output.AssetID == assetID {
				result = append(result, &output.Output)
			}
		}
	}
	return result
}

func (v *MemView) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	return v.ADPs[assetID]
}

func (v *MemView) SaveOutput(o *state.Output) {
	newOutput := &txdb.Output{Output: *o}

	v.Outs[o.Outpoint] = newOutput

	isPayToContract, contractHash, _ := txscript.TestPayToContract(o.TxOutput.Script)
	if isPayToContract {
		_, ok := v.outsByContractHash[*contractHash]
		if ok {
			v.outsByContractHash[*contractHash] = append(v.outsByContractHash[*contractHash], newOutput)
		} else {
			v.outsByContractHash[*contractHash] = outputList{newOutput}
		}
	}
}

func (v *MemView) SaveAssetDefinitionPointer(adp *bc.AssetDefinitionPointer) {
	v.ADPs[adp.AssetID] = adp
}
