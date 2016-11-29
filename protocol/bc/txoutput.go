package bc

import (
	"fmt"
	"io"

	"chain-stealth/crypto/ca"
	"chain-stealth/crypto/sha3pool"
	"chain-stealth/encoding/blockchain"
	"chain-stealth/errors"
)

// TODO(bobg): Review serialization/deserialization logic for
// unknown assetVersions.

type (
	TxOutput struct {
		AssetVersion uint64
		TypedOutput
		ReferenceData []byte
	}

	TypedOutput interface {
		VMVer() uint64
		Program() []byte
		WriteTo(io.Writer) error
		GetAssetAmount() (AssetAmount, bool)
		Amount() (uint64, bool)
		AssetID() (AssetID, bool)
		AssetDescriptor() ca.AssetDescriptor
		ValueDescriptor() ca.ValueDescriptor
		ReadFrom(r io.Reader) error
		writeWitness(io.Writer) error
		readWitness(io.Reader) error
	}
)

func NewTxOutput(assetID AssetID, amount uint64, controlProgram, referenceData []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		TypedOutput: &Outputv1{
			AssetAmount: AssetAmount{
				AssetID: assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
		},
		ReferenceData: referenceData,
	}
}

// NewTxOutputv2 computes a confidential-assets transaction output.
// H is an asset commitment from the inputs being spent or issued.
// c1 is the cumulative blinding factor for H.
// TODO(bobg): What about when multiple inputs are being combined in one output?
func NewTxOutputv2(assetID AssetID, amount uint64, controlProgram, referenceData []byte, rek ca.RecordKey, H ca.AssetCommitment, c1 ca.Scalar) (*TxOutput, *CAValues, error) {
	ad, vd, arp, vrp, c2, f, err := ca.EncryptOutput(rek, ca.AssetID(assetID), amount, 64, []ca.AssetCommitment{H}, c1, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	txout := &TxOutput{
		AssetVersion:  2,
		ReferenceData: referenceData,
		TypedOutput: &Outputv2{
			assetDescriptor: ad,
			valueDescriptor: vd,
			VMVersion:       1,
			ControlProgram:  controlProgram,
			assetRangeProof: arp,
			valueRangeProof: vrp,
		},
	}
	cavals := &CAValues{
		Value:                    amount,
		AssetCommitment:          ad.Commitment(),
		CumulativeBlindingFactor: c2,
		ValueCommitment:          vd.Commitment(),
		ValueBlindingFactor:      f,
	}
	return txout, cavals, nil
}

// EncryptedOutput returns the provided encrypted output as an encrypted
// output, using the provided record key.
func EncryptedOutput(unencrypted *TxOutput, rek ca.RecordKey, prev []ca.AssetCommitment, c ca.Scalar, excess *ca.Scalar) (*TxOutput, *CAValues, error) {
	aa, ok := unencrypted.GetAssetAmount()
	if !ok {
		return nil, nil, errors.New("output already encrypted")
	}

	ad, vd, arp, vrp, c, f, err := ca.EncryptOutput(
		rek,
		ca.AssetID(aa.AssetID),
		aa.Amount,
		64,
		prev,
		c,
		[]byte{0xbe, 0xef},
		excess,
	)
	if err != nil {
		return nil, nil, err
	}

	o := &TxOutput{
		AssetVersion: 2,
		TypedOutput: &Outputv2{
			assetDescriptor: ad,
			valueDescriptor: vd,
			VMVersion:       1,
			ControlProgram:  unencrypted.Program(),
			assetRangeProof: arp,
			valueRangeProof: vrp,
		},
		ReferenceData: unencrypted.ReferenceData,
	}
	v := &CAValues{
		Value:                    aa.Amount,
		AssetCommitment:          ad.Commitment(),
		CumulativeBlindingFactor: c,
		ValueCommitment:          vd.Commitment(),
		ValueBlindingFactor:      f,
	}
	return o, v, nil
}

// assumes r has sticky errors
func (to *TxOutput) readFrom(r io.Reader, txVersion uint64) (err error) {
	to.AssetVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading asset version")
	}
	if (txVersion == 1 && to.AssetVersion != 1) ||
		(txVersion == 2 && to.AssetVersion != 1 && to.AssetVersion != 2) {
		return fmt.Errorf("unrecognized asset version %d for transaction version %d", to.AssetVersion, txVersion)
	}

	to.TypedOutput, err = readOutputCommitment(r, txVersion, to.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "reading output commitment")
	}

	to.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading reference data")
	}

	// TODO(bobg): test that serialization flags include SerWitness, when we relax the serflags-must-be-0x7 rule
	_, err = blockchain.ReadExtensibleString(r, false, func(r io.Reader) error {
		return to.TypedOutput.readWitness(r)
	})
	return err
}

func readOutputCommitment(r io.Reader, txVersion, assetVersion uint64) (result TypedOutput, err error) {
	switch assetVersion {
	case 1:
		result = new(Outputv1)
	case 2:
		result = new(Outputv2)
	default:
		return nil, fmt.Errorf("unrecognized asset version %d", assetVersion)
	}
	all := txVersion == 1 || txVersion == 2
	_, err = blockchain.ReadExtensibleString(r, all, func(r io.Reader) error {
		return result.ReadFrom(r)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "parsing commitment type %T", result)
	}
	return result, nil
}

// assumes r has sticky errors
func (to *TxOutput) writeTo(w io.Writer, serflags byte) error {
	_, err := blockchain.WriteVarint63(w, to.AssetVersion)
	if err != nil {
		return err
	}

	_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
		if to.AssetVersion == 1 || to.AssetVersion == 2 {
			return to.TypedOutput.WriteTo(w)
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = writeRefData(w, to.ReferenceData, serflags)
	if err != nil {
		return err
	}

	if serflags&SerWitness != 0 {
		_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
			return to.TypedOutput.writeWitness(w)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (to *TxOutput) WitnessHash() (hash Hash, err error) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	err = to.TypedOutput.writeWitness(hasher)
	if err != nil {
		return hash, err
	}

	hasher.Read(hash[:])
	return hash, err
}
