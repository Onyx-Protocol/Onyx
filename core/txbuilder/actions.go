package txbuilder

import (
	"bytes"
	"context"
	"crypto/rand"
	stdjson "encoding/json"
	"time"

	"chain-stealth/core/confidentiality"
	"chain-stealth/encoding/json"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/vm"
	"chain-stealth/protocol/vmutil"
)

func ControlProgramActionDecoder(conf *confidentiality.Storage) func(data []byte) (Action, error) {
	// TODO(jackson): Unfortunately, the ControlProgramAction now requires
	// state (for confidentiality key storage). For now, this is just a
	// function that takes the storage and returns an action decoder.
	// Later, we might want a broader txbuilder struct to encapsulate all
	// txbuilder state.

	return func(data []byte) (Action, error) {
		a := new(controlProgramAction)
		err := stdjson.Unmarshal(data, a)
		if err != nil {
			return nil, err
		}
		a.confidentiality = conf
		return a, nil
	}
}

type controlProgramAction struct {
	confidentiality *confidentiality.Storage
	bc.AssetAmount
	Program            json.HexBytes `json:"control_program"`
	ConfidentialityKey json.HexBytes `json:"confidentiality_key"`
	ReferenceData      json.Map      `json:"reference_data"`
}

func (a *controlProgramAction) Build(ctx context.Context, maxTime time.Time, b *TemplateBuilder) error {
	var missing []string
	if len(a.Program) == 0 {
		missing = append(missing, "control_program")
	}
	if len(a.ConfidentialityKey) == 0 {
		missing = append(missing, "confidentiality_key")
	}
	if a.AssetID == (bc.AssetID{}) {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}
	if len(a.ConfidentialityKey) != 32 {
		return ErrBadConfidentialityKey
	}

	// Verify that there is no existing stored confidentiality key for
	// this control program, or if there is, that it exactly matches the
	// provided confidentiality key.
	keys, err := a.confidentiality.GetKeys(ctx, [][]byte{a.Program})
	if err != nil {
		return err
	}
	if len(keys) == 1 && !bytes.Equal(a.ConfidentialityKey, keys[0].Key[:]) {
		return ErrBadConfidentialityKey
	}

	// Store the confidentiality key so that we can unblind the output
	// when it's confirmed.
	key := &confidentiality.Key{ControlProgram: a.Program}
	copy(key.Key[:], a.ConfidentialityKey)
	b.OnBuild(func() error { return a.confidentiality.StoreKeys(ctx, []*confidentiality.Key{key}) })

	out := bc.NewTxOutput(a.AssetID, a.Amount, a.Program, a.ReferenceData)
	return b.AddEncryptedOutput(out, key.Key)
}

func DecodeSetTxRefDataAction(data []byte) (Action, error) {
	a := new(setTxRefDataAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type setTxRefDataAction struct {
	Data json.Map `json:"reference_data"`
}

func (a *setTxRefDataAction) Build(ctx context.Context, maxTime time.Time, b *TemplateBuilder) error {
	if len(a.Data) == 0 {
		return MissingFieldsError("reference_data")
	}
	return b.setReferenceData(a.Data)
}

func RetireActionDecoder(conf *confidentiality.Storage) func(data []byte) (Action, error) {
	return func(data []byte) (Action, error) {
		a := new(retireAction)
		err := stdjson.Unmarshal(data, a)
		if err != nil {
			return nil, err
		}
		a.confidentiality = conf
		return a, nil
	}
}

type retireAction struct {
	confidentiality *confidentiality.Storage
	bc.AssetAmount
	ReferenceData json.Map `json:"reference_data"`
}

func (a *retireAction) Build(ctx context.Context, maxTime time.Time, b *TemplateBuilder) error {
	// Create a unique retirement control program so that we can store
	// an association between the confidentiality key and the action.
	var retirementNonce [32]byte
	_, err := rand.Read(retirementNonce[:])
	if err != nil {
		return err
	}
	prog := vmutil.NewBuilder().
		AddOp(vm.OP_FAIL).
		AddData(retirementNonce[:]).
		AddOp(vm.OP_DROP).
		Program

	// Create a new control program and store the association with the
	// control program.
	k, err := confidentiality.NewKey()
	if err != nil {
		return err
	}
	err = a.confidentiality.StoreKeys(ctx, []*confidentiality.Key{{Key: k, ControlProgram: prog}})
	if err != nil {
		return err
	}

	out := bc.NewTxOutput(a.AssetID, a.Amount, prog, a.ReferenceData)
	return b.AddEncryptedOutput(out, k)
}

func DecodeRawTransactionAction(data []byte) (Action, error) {
	a := new(rawTransactionAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type rawTransactionAction struct {
	Transaction         *bc.TxData            `json:"raw_transaction"`
	SigningInstructions []*SigningInstruction `json:"signing_instructions"`
}

func (a *rawTransactionAction) Build(ctx context.Context, maxTime time.Time, b *TemplateBuilder) error {
	if a.Transaction == nil {
		return MissingFieldsError("raw_transaction")
	}

	// Add the same mintime, maxtime restrictions.
	if a.Transaction.MinTime != 0 {
		b.RestrictMinTimeMS(a.Transaction.MinTime)
	}
	if a.Transaction.MaxTime != 0 {
		b.RestrictMaxTimeMS(a.Transaction.MaxTime)
	}

	// Add the signing instructions. Note that since they're added to TemplateBuilder,
	// they are not finalized. When the TemplateBuilder builds a Template, the signing
	// instructions will be updated to new, correct, positions.
	unsigned := make(map[int]*SigningInstruction)
	for _, si := range a.SigningInstructions {
		unsigned[si.Position] = si
	}

	// If the outputs are confidential, the client should
	// have included their `confidentiality_instructions` in the build request,
	// so we'll know how to balance blinding factors appropriately.
	for pos, in := range a.Transaction.Inputs {
		if sigInst, ok := unsigned[pos]; ok {
			err := b.AddRawInput(in, sigInst)
			if err != nil {
				return err
			}
		} else {
			// If there's no signing instruction, assume it's already
			// been signed and add it w/o any instructions.
			b.signedInputs = append(b.signedInputs, in)
		}
	}
	for _, o := range a.Transaction.Outputs {
		err := b.AddRawOutput(o)
		if err != nil {
			return err
		}
	}

	return nil
}
