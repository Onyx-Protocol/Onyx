package bc

import (
	"bytes"
	"io"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/encoding/blockchain"
)

// TODO(bobg): Review serialization/deserialization logic for
// assetVersions other than 1.

type (
	TxInput struct {
		AssetVersion uint32
		InputCommitment
		ReferenceData []byte
		InputWitness  [][]byte
	}

	InputCommitment interface {
		IsIssuance() bool
		readFrom(io.Reader, uint32) error
		writeTo(io.Writer, uint32, uint8)
	}

	SpendInputCommitment struct {
		Outpoint
		OutputCommitment
	}

	IssuanceInputCommitment struct {
		MinTimeMS, MaxTimeMS uint64
		InitialBlock         Hash
		Amount               uint64
		VMVersion            uint32
		IssuanceProgram      []byte
		AssetDefinition      []byte
	}
)

func NewSpendInput(txhash Hash, index uint32, inputWitness [][]byte, assetID AssetID, amount uint64, controlProgram, referenceData []byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		InputCommitment: &SpendInputCommitment{
			Outpoint: Outpoint{
				Hash:  txhash,
				Index: index,
			},
			OutputCommitment: OutputCommitment{
				AssetAmount: AssetAmount{
					AssetID: assetID,
					Amount:  amount,
				},
				VMVersion:      1,
				ControlProgram: controlProgram,
			},
		},
		ReferenceData: referenceData,
		InputWitness:  inputWitness,
	}
}

func NewIssuanceInput(minTime, maxTime time.Time, initialBlock Hash, amount uint64, issuanceProgram, assetDefinition, referenceData []byte, inputWitness [][]byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		InputCommitment: &IssuanceInputCommitment{
			MinTimeMS:       uint64(minTime.UnixNano()) / uint64(time.Millisecond),
			MaxTimeMS:       uint64(maxTime.UnixNano()) / uint64(time.Millisecond),
			InitialBlock:    initialBlock,
			Amount:          amount,
			VMVersion:       1,
			IssuanceProgram: issuanceProgram,
			AssetDefinition: assetDefinition,
		},
		ReferenceData: referenceData,
		InputWitness:  inputWitness,
	}
}

func (t TxInput) AssetAmount() AssetAmount {
	if ic, ok := t.InputCommitment.(*IssuanceInputCommitment); ok {
		return AssetAmount{
			AssetID: ic.AssetID(),
			Amount:  ic.Amount,
		}
	}
	sc := t.InputCommitment.(*SpendInputCommitment)
	return sc.AssetAmount
}

func (t TxInput) AssetID() AssetID {
	if ic, ok := t.InputCommitment.(*IssuanceInputCommitment); ok {
		return ic.AssetID()
	}
	sc := t.InputCommitment.(*SpendInputCommitment)
	return sc.AssetID
}

func (t TxInput) Amount() uint64 {
	if ic, ok := t.InputCommitment.(*IssuanceInputCommitment); ok {
		return ic.Amount
	}
	sc := t.InputCommitment.(*SpendInputCommitment)
	return sc.Amount
}

func (t TxInput) AssetDefinition() []byte {
	if ic, ok := t.InputCommitment.(*IssuanceInputCommitment); ok {
		return ic.AssetDefinition
	}
	return nil
}

func (t TxInput) ControlProgram() []byte {
	if sc, ok := t.InputCommitment.(*SpendInputCommitment); ok {
		return sc.ControlProgram
	}
	return nil
}

func (t *TxInput) readFrom(r io.Reader) (err error) {
	assetVersion, _ := blockchain.ReadUvarint(r)
	t.AssetVersion = uint32(assetVersion)
	inputCommitment, err := blockchain.ReadBytes(r, commitmentMaxByteLength) // TODO(bobg): is this the right max?
	if err != nil {
		return err
	}
	var (
		ic *IssuanceInputCommitment
		sc *SpendInputCommitment
	)
	if assetVersion == 1 {
		icBuf := bytes.NewBuffer(inputCommitment)
		var icType [1]byte
		_, err = io.ReadFull(icBuf, icType[:])
		if err != nil {
			return err
		}
		switch icType[0] {
		case 0:
			ic = new(IssuanceInputCommitment)
			err = ic.readFrom(icBuf, t.AssetVersion)
			if err != nil {
				return err
			}
			t.InputCommitment = ic
		case 1:
			sc = new(SpendInputCommitment)
			err = sc.readFrom(icBuf, t.AssetVersion)
			if err != nil {
				return err
			}
			t.InputCommitment = sc
		}
	}
	t.ReferenceData, _ = blockchain.ReadBytes(r, metadataMaxByteLength)
	inputWitness, err := blockchain.ReadBytes(r, witnessMaxByteLength)
	if err != nil {
		return err
	}
	if assetVersion == 1 {
		iwBuf := bytes.NewBuffer(inputWitness)
		n, err := blockchain.ReadUvarint(iwBuf)
		if err != nil {
			return err
		}
		for n > 0 {
			arg, err := blockchain.ReadBytes(iwBuf, witnessMaxByteLength)
			if err != nil {
				return err
			}
			t.InputWitness = append(t.InputWitness, arg)
			n--
		}
	}
	return nil
}

func (t TxInput) writeTo(w io.Writer, serflags uint8) {
	blockchain.WriteUvarint(w, uint64(t.AssetVersion))
	t.InputCommitment.writeTo(w, t.AssetVersion, serflags)
	writeMetadata(w, t.ReferenceData, serflags)
	if serflags&SerWitness != 0 {
		blockchain.WriteBytes(w, t.inputWitnessBytes())
	}
}

func (t TxInput) InputCommitmentBytes(serflags uint8) []byte {
	var b bytes.Buffer
	t.InputCommitment.writeTo(&b, t.AssetVersion, serflags)
	return b.Bytes()
}

func (t TxInput) inputWitnessBytes() []byte {
	var b bytes.Buffer
	blockchain.WriteUvarint(&b, uint64(len(t.InputWitness)))
	for _, arg := range t.InputWitness {
		blockchain.WriteBytes(&b, arg)
	}
	return b.Bytes()
}

func (t TxInput) WitnessHash() Hash {
	return sha3.Sum256(t.inputWitnessBytes())
}

func (t TxInput) Outpoint() (o Outpoint) {
	if sc, ok := t.InputCommitment.(*SpendInputCommitment); ok {
		o = sc.Outpoint
	}
	return o
}

func (sc SpendInputCommitment) IsIssuance() bool { return false }

// Parsing within the Extensible String; the spend/issuance byte has
// already been consumed.
func (sc *SpendInputCommitment) readFrom(r io.Reader, assetVersion uint32) error {
	sc.Outpoint.readFrom(r)
	sc.OutputCommitment.readFrom(r, assetVersion)
	return nil
}

// Writing the Extensible String, including the spend/issuance byte.
func (sc SpendInputCommitment) writeTo(w io.Writer, assetVersion uint32, serflags uint8) {
	var b bytes.Buffer
	b.Write([]byte{1}) // "spend" type
	sc.Outpoint.WriteTo(&b)
	if serflags&SerPrevout != 0 {
		sc.OutputCommitment.writeTo(&b, assetVersion)
	}
	blockchain.WriteBytes(w, b.Bytes())
}

func (ic IssuanceInputCommitment) AssetID() AssetID {
	return ComputeAssetID(ic.IssuanceProgram, ic.InitialBlock, ic.VMVersion)
}

func (ic IssuanceInputCommitment) IsIssuance() bool { return true }

// Parsing within the Extensible String; the spend/issuance byte has
// already been consumed.
func (ic *IssuanceInputCommitment) readFrom(r io.Reader, _ uint32) (err error) {
	ic.MinTimeMS, _ = blockchain.ReadUvarint(r)
	ic.MaxTimeMS, _ = blockchain.ReadUvarint(r)
	io.ReadFull(r, ic.InitialBlock[:])
	ic.Amount, _ = blockchain.ReadUvarint(r)
	v, _ := blockchain.ReadUvarint(r)
	ic.VMVersion = uint32(v)
	ic.IssuanceProgram, err = blockchain.ReadBytes(r, MaxProgramByteLength)
	if err != nil {
		return err
	}
	ic.AssetDefinition, err = blockchain.ReadBytes(r, assetDefinitionMaxByteLength)
	if err != nil {
		return err
	}
	return nil
}

// Writing the Extensible String, including the spend/issuance byte.
func (ic IssuanceInputCommitment) writeTo(w io.Writer, _ uint32, serflags uint8) {
	var b bytes.Buffer
	b.Write([]byte{0}) // "issuance" type
	blockchain.WriteUvarint(&b, ic.MinTimeMS)
	blockchain.WriteUvarint(&b, ic.MaxTimeMS)
	b.Write(ic.InitialBlock[:])
	blockchain.WriteUvarint(&b, ic.Amount)
	blockchain.WriteUvarint(&b, uint64(ic.VMVersion))
	blockchain.WriteBytes(&b, ic.IssuanceProgram)
	blockchain.WriteBytes(&b, ic.AssetDefinition)
	writeMetadata(w, b.Bytes(), serflags)
}
