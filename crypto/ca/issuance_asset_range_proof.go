package ca

import (
	"fmt"
	"io"

	"chain-stealth/encoding/blockchain"
)

type IssuanceAssetRangeProof struct {
	rs      *ringSignature
	Y       []Point // issuance keys
	vmver   uint64
	program []byte
	args    [][]byte
}

func (iarp *IssuanceAssetRangeProof) VMVersion() uint64 {
	return iarp.vmver
}

func (iarp *IssuanceAssetRangeProof) Program() []byte {
	return iarp.program
}

func (iarp *IssuanceAssetRangeProof) Arguments() [][]byte {
	return iarp.args
}

func (iarp *IssuanceAssetRangeProof) SetArguments(args [][]byte) {
	iarp.args = args
}

func (iarp *IssuanceAssetRangeProof) WriteTo(w io.Writer) error {
	err := iarp.rs.writeTo(w)
	if err != nil {
		return err
	}
	for _, y := range iarp.Y {
		err = y.WriteTo(w)
		if err != nil {
			return err
		}
	}
	_, err = blockchain.WriteVarint63(w, iarp.vmver)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, iarp.program)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstrList(w, iarp.args)
	return err
}

func (iarp *IssuanceAssetRangeProof) ReadFrom(r io.Reader, n uint32) error {
	iarp.rs = new(ringSignature)
	err := iarp.rs.readFrom(r, n)
	if err != nil {
		return err
	}
	iarp.Y = make([]Point, iarp.rs.nPubkeys())
	for i := range iarp.Y {
		err = iarp.Y[i].readFrom(r)
		if err != nil {
			return err
		}
	}
	iarp.vmver, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return err
	}
	iarp.program, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}
	iarp.args, _, err = blockchain.ReadVarstrList(r)
	return err
}

// TODO(bobg): This package is the wrong place for VM-related concepts
// like vmversion and signature programs, move these functions
// elsewhere.

func CreateIssuanceAssetRangeProof(
	H AssetCommitment,
	c Scalar, // the cumulative blinding factor for commitment `H` such that: `H = A[j] + c·G`.
	a []AssetID,
	Y []Point, // issuance keys
	vmver uint64, // VM version for the signature program
	program []byte, // signature program that authenticates the transaction
	j int, // index of the asset being issued: `H = A[j] + c*G`.
	y Scalar, // private key for the issuance key corresponding to the asset being issued: `Y[j] = y·G`.
) (*IssuanceAssetRangeProof, error) {
	n := len(a)
	if n == 0 {
		return nil, fmt.Errorf("list of non-blinded asset IDs {A} is empty")
	}
	if len(Y) != n {
		return nil, fmt.Errorf("lists of non-blinded asset IDs {A} and issuance keys {Y} are not of the same length: len(A)=%d vs len(Y)=%d", len(a), len(Y))
	}
	if j < 0 || j >= n {
		return nil, fmt.Errorf("designated index j is out of bounds: %d not in [0, %d]", j, len(a)-1)
	}

	// 1. Calculate non-blinded asset commitments for the values in a: `A[i] = 8·Decode(SHA3(a[i]))`.
	A := make([]AssetCommitment, n)
	for i, assetID := range a {
		A[i] = CreateNonblindedAssetCommitment(assetID)
	}

	// 2. Calculate a 96-byte commitment string: `commit = SHAKE256(0x66 || H || A[0] || ... || A[n-1] || Y[0] || ... || Y[n-1] || vmver || program, 8*96)`.
	commit := shake256([]byte{0x66}, H.Bytes())
	for i := 0; i < n; i++ {
		commit.Write(A[i].Bytes())
	}
	for i := 0; i < n; i++ {
		Ybytes := encodePoint(&Y[i])
		commit.Write(Ybytes[:])
	}
	commit.Write(uint64le(vmver))
	commit.Write(program)

	// 3. Calculate message to sign as first 32 bytes of the commitment string: `msg = commit[0:32]`.
	var msg [32]byte
	commit.Read(msg[:])

	// 4. Calculate the coefficient `h` from the remaining 64 bytes of the commitment string: `h = commit[32:96]`.
	//    Interpret `h` as a 64-byte little-endian integer and reduce modulo subgroup order `L`.
	var hbuf [64]byte
	commit.Read(hbuf[:])
	h := reducedScalar(hbuf)

	// 5. Calculate `n` public keys `{P[i]}`: `P[i] = H - A[i] + h·Y[i]`.
	P := calcIARPPubkeys(n, H, A, h, Y)

	// 6. Calculate private key `p = c + h·y`.
	p := multiplyAndAddScalars(h, y, c)

	// 7. Create a ring signature with:
	//     * message `msg`,
	//     * `n` public keys `{P[i]}`,
	//     * index `j`,
	//     * private key `p`.
	rs := createRingSignature(msg, P, j, p)

	// 8. Return an issuance range proof consisting of `(e0,{s[i]}, {Y[i]}, vmver, program, [])`.
	iarp := new(IssuanceAssetRangeProof)
	iarp.rs = rs
	iarp.Y = Y
	iarp.vmver = vmver
	iarp.program = program
	return iarp, nil
}

func (iarp *IssuanceAssetRangeProof) Verify(
	H AssetCommitment, // H, the asset ID commitment
	a []AssetID,
) error {
	n := len(a)
	if n == 0 {
		return fmt.Errorf("empty assetID list")
	}
	if len(iarp.Y) != n {
		return fmt.Errorf("number of issuance keys %d does not match length of assetID list %d", len(iarp.Y), n)
	}

	// 1. Calculate non-blinded asset commitments for the values in a: `A[i] = 8·Decode(SHA3(a[i]))`.
	A := make([]AssetCommitment, n)
	for i, assetID := range a {
		A[i] = CreateNonblindedAssetCommitment(assetID)
	}

	// 2. Calculate a 96-byte commitment string: `commit = SHAKE256(0x66 || H || A[0] || ... || A[n-1] || Y[0] || ... || Y[n-1] || vmver || program, 8*96)`.
	commit := shake256([]byte{0x66}, H.Bytes())
	for i := 0; i < n; i++ {
		commit.Write(A[i].Bytes())
	}
	for i := 0; i < n; i++ {
		Ybytes := encodePoint(&iarp.Y[i])
		commit.Write(Ybytes[:])
	}
	commit.Write(uint64le(iarp.vmver))
	commit.Write(iarp.program)

	// 3. Calculate message to sign as first 32 bytes of the commitment string: `msg = commit[0:32]`.
	var msg [32]byte
	commit.Read(msg[:])

	// 4. Calculate the coefficient `h` from the remaining 64 bytes of the commitment string: `h = commit[32:96]`.
	//    Interpret `h` as a 64-byte little-endian integer and reduce modulo subgroup order `L`.
	var hbuf [64]byte
	commit.Read(hbuf[:])
	h := reducedScalar(hbuf)

	// 5. Calculate the `n` public keys `{P[i]}`: `P[i] = H - A[i] + h*Y[i]`.
	P := calcIARPPubkeys(n, H, A, h, iarp.Y)

	// 6. [Verify the ring signature](#verify-ring-signature) `e[0], s[0], ... s[n-1]` with message `msg` and public keys `{P[i]}`.
	return iarp.rs.verify(msg, P)
}

func calcIARPPubkeys(n int, H AssetCommitment, A []AssetCommitment, h Scalar, Y []Point) []Point {
	P := make([]Point, n)
	for i := 0; i < n; i++ {
		P[i] = Point(H)
		P[i].sub((*Point)(&A[i]))
		hY := Y[i]
		hY.mul(&h)
		P[i].add(&hY)
	}
	return P
}
