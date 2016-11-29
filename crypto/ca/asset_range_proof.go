package ca

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"chain-stealth/crypto/sha3pool"
	"chain-stealth/encoding/blockchain"
)

type AssetRangeProof struct {
	H  []AssetCommitment
	rs *ringSignature
}

func CreateAssetRangeProof(H AssetCommitment, ea EncryptedAssetID, candidates []AssetCommitment, j int, bf Scalar) (*AssetRangeProof, error) {
	// FIXME: add sanity checks that all lengths are in bounds and match.

	// Calculate the message to sign: `msg = SHA3-256(0x55 || H’ || H[0] || ... || H[n-1] || ea || ec)`.
	msg := calcAssetRangeProofMsg(H, ea, candidates)

	// Calculate the set of public keys for the ring signature from the set of input asset ID commitments: `P[i] = H’ - H[i]`.
	pubkeys := calcAssetRangeProofPubkeys(H, candidates)

	// (Calculate the private key: `p = d`)
	// Create a ring signature using `msg`, `{P[i]}`, `j`, and `p`.
	rs := createRingSignature(msg, pubkeys, j, bf)

	// Return the list of asset ID commitments `{H[i]}` and the ring signature `e[0], s[0], ... s[n-1]`.
	return &AssetRangeProof{H: candidates, rs: rs}, nil
}

func (arp *AssetRangeProof) Verify(
	H AssetCommitment,
	eaec *EncryptedAssetID,
) error {
	// FIXME: add sanity checks that all lengths are in bounds and match each other.

	if eaec == nil {
		eaec = &EncryptedAssetID{}
	}

	// Calculate `msg = SHA3-256(0x55 || H’ || H[0] || ... || H[n-1] || ea || ec)`.
	msg := calcAssetRangeProofMsg(H, *eaec, arp.H)

	// Calculate the set of public keys for the ring signature from the set of input asset ID commitments: `P[i] = H’ - H[i]`.
	pubkeys := calcAssetRangeProofPubkeys(H, arp.H)

	// Verify the ring signature `e[0], s[0], ... s[n-1]` with `msg` and `{P[i]}`.
	// Return true if verification was successful, and false otherwise.
	return arp.rs.verify(msg, pubkeys)
}

func calcAssetRangeProofMsg(H AssetCommitment, ea EncryptedAssetID, candidates []AssetCommitment) (msg [32]byte) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)

	h.Write([]byte{0x55})
	h.Write(H.Bytes())
	for _, c := range candidates {
		h.Write(c.Bytes())
	}
	h.Write(ea.AssetID[:])
	h.Write(ea.BlindingFactor[:])

	h.Read(msg[:])
	return msg
}

func calcAssetRangeProofPubkeys(H AssetCommitment, candidates []AssetCommitment) []Point {
	// Calculate the set of public keys for the ring signature from the set of input asset ID commitments: `P[i] = H’ - H[i]`.
	pubkeys := make([]Point, len(candidates))
	for i, c := range candidates {
		pubkeys[i] = Point(H)
		pubkeys[i].sub((*Point)(&c))
	}
	return pubkeys
}

func (rp *AssetRangeProof) WriteTo(w io.Writer) error {
	_, err := blockchain.WriteVarint31(w, uint64(len(rp.H)))
	if err != nil {
		return err
	}
	for _, h := range rp.H {
		err = h.writeTo(w)
		if err != nil {
			return err
		}
	}
	return rp.rs.writeTo(w)
}

func (rp *AssetRangeProof) ReadFrom(r io.Reader) error {
	n, _, err := blockchain.ReadVarint31(r)
	if err != nil {
		return err
	}
	rp.H = make([]AssetCommitment, n)
	for i := uint32(0); i < n; i++ {
		err = rp.H[i].readFrom(r)
		if err != nil {
			return err
		}
	}
	rp.rs = new(ringSignature)
	return rp.rs.readFrom(r, n)
}

func (rp *AssetRangeProof) String() string {
	Hstrs := make([]string, 0, len(rp.H))
	for _, H := range rp.H {
		Hstrs = append(Hstrs, hex.EncodeToString(H.Bytes()))
	}
	return fmt.Sprintf("{AssetCommitments: [%s]; RingSignature %s}", strings.Join(Hstrs, " "), rp.rs)
}
