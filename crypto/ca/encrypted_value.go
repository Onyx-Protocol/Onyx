package ca

import (
	"encoding/binary"
	"fmt"
	"io"
)

type EncryptedValue struct {
	Value          [8]byte
	BlindingFactor [32]byte
}

func (ev *EncryptedValue) readFrom(r io.Reader) error {
	_, err := io.ReadFull(r, ev.Value[:])
	if err != nil {
		return err
	}
	_, err = io.ReadFull(r, ev.BlindingFactor[:])
	return err
}

func (ev *EncryptedValue) WriteTo(w io.Writer) error {
	_, err := w.Write(ev.Value[:])
	if err != nil {
		return err
	}
	_, err = w.Write(ev.BlindingFactor[:])
	return err
}

func EncryptValue(vc ValueCommitment, value uint64, bf Scalar, vek ValueKey) EncryptedValue {
	// Expand the encryption key: `ek = SHA3-512(vek || V)`, split the resulting hash in two halves.
	ek := hash512(vek[:], vc.Bytes())

	// Encrypt the value using the first half: `ev = value XOR ek[0,8]`.
	ev := xor64(uint64le(value), ek[0:8])

	// Encrypt the value blinding factor using the second half: `ef = f XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
	ef := xor256(bf[:], ek[8:40])

	// Return `(ev, ef)`.
	return EncryptedValue{
		Value:          ev,
		BlindingFactor: ef,
	}
}

func (ev EncryptedValue) Decrypt(vc ValueCommitment, ac AssetCommitment, vek ValueKey) (value uint64, bf Scalar, ok bool) {
	// Expand the encryption key: `ek = SHA3-512(vek || V)`, split the resulting hash in two halves.
	// TODO(bobg): factor out common code with EncryptValue above
	ek := hash512(vek[:], vc.Bytes())

	// Decrypt the value using the first half: `value = ev XOR ek[0,8]`.
	valueBytes := xor64(ev.Value[:], ek[0:8])
	value = binary.LittleEndian.Uint64(valueBytes[:])

	// Decrypt the value blinding factor using the second half: `f = ef XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
	bf = xor256(ev.BlindingFactor[:], ek[8:40])

	// Calculate `P = value*H’ + f*G`.
	P := multiplyAndAddPoint(scalarFromUint64(value), Point(ac), bf)

	// Verify that `P` equals `V`. If not, halt and return `nil`.
	vcp := Point(vc)
	if !P.equal(&vcp) {
		return 0, bf, false
	}

	// Return `(value, f)`.
	return value, bf, true
}

func (ev EncryptedValue) String() string {
	return fmt.Sprintf("{Value: %d; BlindingFactor: %x}", binary.LittleEndian.Uint64(ev.Value[:]), ev.BlindingFactor[:])
}
