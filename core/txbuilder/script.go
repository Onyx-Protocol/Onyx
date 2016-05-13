package txbuilder

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
)

type scriptReceiver struct {
	script []byte
}

func (receiver *scriptReceiver) PKScript() []byte { return receiver.script }

func newScriptReceiver(script []byte) *scriptReceiver {
	return &scriptReceiver{
		script: script,
	}
}

// NewScriptDestination returns a Destination
// that will use the supplied script
// as the PKScript in the output
func NewScriptDestination(ctx context.Context, assetAmount *bc.AssetAmount, script []byte, metadata []byte) *Destination {
	scriptReceiver := newScriptReceiver(script)
	dest := &Destination{
		AssetAmount: *assetAmount,
		Metadata:    metadata,
		Receiver:    scriptReceiver,
	}
	return dest
}
