package asset

import (
	"golang.org/x/net/context"

	"chain/api/txbuilder"
	"chain/fedchain/bc"
)

type ScriptReceiver struct {
	script []byte
}

func (receiver *ScriptReceiver) PKScript() []byte { return receiver.script }

func NewScriptReceiver(script []byte) *ScriptReceiver {
	return &ScriptReceiver{
		script: script,
	}
}

func NewScriptDestination(ctx context.Context, assetAmount *bc.AssetAmount, script []byte, metadata []byte) (*txbuilder.Destination, error) {
	scriptReceiver := NewScriptReceiver(script)
	dest := &txbuilder.Destination{
		AssetAmount: *assetAmount,
		Metadata:    metadata,
		Receiver:    scriptReceiver,
	}
	return dest, nil
}
