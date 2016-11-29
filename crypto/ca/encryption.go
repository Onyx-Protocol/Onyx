package ca

import (
	"bytes"
	"errors"
	"fmt"

	"chain-stealth/encoding/blockchain"
	"chain-stealth/encoding/bufpool"
)

// TODO(bobg): This package is the wrong place for VM-related concepts
// like vmversion and signature programs, move these functions
// elsewhere.

var ErrCannotDecrypt = errors.New("cannot decrypt")

// Inputs:
// 1. `rek`: the [record encryption key](#record-encryption-key) unique to this issuance.
// 2. `assetID`: the output asset ID.
// 3. `value`: the output amount.
// 4. `N`: number of bits to encrypt (`value` must fit within `N` bits).
// 5. `{(assetid[i], Y[i])}`: `n` input asset IDs and corresponding issuance public keys.
// 6. `y`: issuance key for `assetID` such that `Y[j] = y·G` where `j` is the index of the issued asset: `a[j] == assetID`.
// 7. `(vmver’,program’)`: the signature program and its VM version to be signed by the issuance proof.
//
// Outputs:
// 1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
// 2. `VD`: the [value descriptor](#value-descriptor).
// 3. `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
// 4. `VRP`: the [value range proof](#value-range-proof).
// 5. `c`: the [cumulative blinding factor](#cumulative-blinding-factor) for the asset ID commitment `H`.
// 6. `f`: the [value blinding factor](#value-blinding-factor).
func EncryptIssuance(
	rek RecordKey,
	assetID AssetID,
	value uint64,
	N uint8,
	assetids []AssetID,
	Y []Point,
	y Scalar,
	vmver uint64,
	program []byte,
) (
	AD AssetDescriptor,
	VD ValueDescriptor,
	iarp *IssuanceAssetRangeProof,
	vrp *ValueRangeProof,
	c Scalar,
	f Scalar,
	err error,
) {
	// 1. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)

	// 2. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
	vek := DeriveValueKey(iek)

	// 4. Find `j` index of the `assetID` among `{assetid[i]}`. If not found, halt and return `nil`.
	j := -1
	for i, aid := range assetids {
		if aid == assetID {
			j = i
			break
		}
	}
	if j == -1 {
		err = fmt.Errorf("asset ID is not found among array of declared asset IDs")
		return
	}

	// 5. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment): compute `(H,d)` from `(A, 0, aek)`.
	// 6. Set `c = d`.
	A := CreateNonblindedAssetCommitment(assetID)
	H, c := CreateBlindedAssetCommitment(A, ZeroScalar, aek)

	// 7. [Create Blinded Value Commitment](#create-blinded-value-commitment): compute `(V,f)` from `(vek, value, H, c)`.
	V, f := CreateBlindedValueCommitment(vek, value, H)

	// 8. [Create Issuance Asset Range Proof](#create-issuance-asset-range-proof): compute `IARP` from `(H, c, {a[i]}, {Y[i]}, vmver’, program’, j, y)`.
	iarp, err = CreateIssuanceAssetRangeProof(H, c, assetids, Y, vmver, program, j, y)
	if err != nil {
		return
	}

	// 9. [Create Value Range Proof](#create-value-range-proof): compute `VRP` from `(H, V, (0x00...,0x00...), N, value, {0x00...}, f, rek)`.
	pt := make([][32]byte, 2*N-1)
	vrp, err = CreateValueRangeProof(H, V, EncryptedValue{}, N, value, pt, f, rek)

	// 10. Create [blinded asset ID descriptor](#blinded-asset-id-descriptor) `AD` containing `H` and all-zero [encrypted asset ID](#encrypted-asset-id).
	AD = &BlindedAssetDescriptor{H: H}

	// 11. Create [blinded value descriptor](#blinded-value-descriptor) `VD` containing `V` and all-zero [encrypted value](#encrypted-value).
	VD = &BlindedValueDescriptor{V: V}

	// 12. Return `(AD, VD, IARP, VRP, c, f)`.
	return
}

// Inputs:
// 1. `rek`: the [record encryption key](#record-encryption-key).
// 2. `assetID`: the output asset ID.
// 3. `value`: the output amount.
// 4. `N`: number of bits to encrypt (`value` must fit within `N` bits).
// 5. `{H[i]}`: `n` input [asset ID commitments](#asset-id-commitment).
// 6. `c`: input [cumulative blinding factor](#cumulative-blinding-factor) corresponding to one of the asset ID commitments `{H[i]}`.
// 7. `plaintext`: binary string that has length of less than `32·(2·N-1)` bytes when encoded as [varstring31](data.md#varstring31).
// 8. Optional `q`: the [excess factor](#excess-commitment) to have this output balance with the transaction. If omitted, blinding factor is generated at random.
//
// Outputs:
// 1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
// 2. `VD`: the [value descriptor](#value-descriptor).
// 3. `ARP`: the [asset ID range proof](#asset-range-proof).
// 4. `VRP`: the [value range proof](#value-range-proof).
// 5. `c’`: the output [cumulative blinding factor](#cumulative-blinding-factor) for the asset ID commitment `H’`.
// 6. `f’`: the output [value blinding factor](#value-blinding-factor).
func EncryptOutput(
	rek RecordKey,
	assetID AssetID,
	amount uint64,
	N uint8,
	H []AssetCommitment,
	cprev Scalar,
	plaintext []byte,
	q *Scalar,
) (
	AD AssetDescriptor,
	VD ValueDescriptor,
	arp *AssetRangeProof,
	vrp *ValueRangeProof,
	c Scalar,
	f Scalar,
	err error,
) {
	// 4. If `value ≥ 2^N`, halt and return `nil`.
	if amount >= 1<<N {
		err = fmt.Errorf("value %d does not fit into %d bits", amount, N)
	}

	// 1. Encode `plaintext` using [varstring31](data.md#varstring31) encoding and split the string in 32-byte chunks `{pt[i]}` (last chunk padded with zero bytes if needed).
	ptbuf := bufpool.Get()
	defer bufpool.Put(ptbuf)
	_, err = blockchain.WriteVarstr31(ptbuf, plaintext)
	if err != nil {
		return
	}
	// 3. If the number of chunks `{pt[i]}` is less than `2·N-1`, pad the array with all-zero 32-byte chunks.
	ptchunks := make([][32]byte, 2*N-1)
	ptrdr := bytes.NewReader(ptbuf.Bytes())
	for i := 0; i < len(ptchunks); i++ {
		n, _ := ptrdr.Read(ptchunks[i][:])
		if n < 32 {
			break
		}
	}
	if ptrdr.Len() > 0 {
		err = fmt.Errorf("plaintext too long")
		return
	}

	// 5. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)

	// 6. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
	vek := DeriveValueKey(iek)

	// 7. [Create Nonblinded Asset ID Commitment](#create-nonblinded-asset-id-commitment): compute `A` from `assetID`.
	A := CreateNonblindedAssetCommitment(assetID)

	// 8. Find index `j` among `{H[i]}` such that `H[j] == A + c[j]·G`. If index cannot be found, halt and return `nil`.
	j := -1
	for i, Hpoint := range H {
		Hcandidate := multiplyAndAddPoint(one, Point(A), cprev)
		if Hpoint == AssetCommitment(Hcandidate) {
			j = i
			break
		}
	}
	if j == -1 {
		err = fmt.Errorf("asset ID is not found among array of previous asset ID commitments")
		return
	}

	// 9. [Create Blinded Asset ID Commitment](#create-blinded-asset-id-commitment): compute `(H’,d)` from `(H[j], c[j], aek)`.
	// 10. Calculate `c’ = c[j] + d mod L`.
	var d Scalar
	Hprime, d := CreateBlindedAssetCommitment(H[j], cprev, aek)
	c = cprev
	c.Add(&d)

	// 11. [Encrypt Asset ID](#encrypt-asset-id): compute `(ea,ec)` from `(assetID, H’, c’, aek)`.
	ea := EncryptAssetID(assetID, Hprime, c, aek)

	// 12. [Create Blinded Value Commitment](#create-blinded-value-commitment): compute `(V’,f’)` from `(vek, value, H’, c’)`.
	V, f := CreateBlindedValueCommitment(vek, amount, Hprime)

	// 13. If `q` is provided:
	if q != nil {
		// 1. Compute `extra` scalar: `extra = q - f’ - value·c’`.
		var extra Scalar
		extra = *q
		extra.sub(&f)
		valueMulC := scalarFromUint64(amount)
		valueMulC.mulAdd(&c, &ZeroScalar)
		extra.sub(&valueMulC)

		// 2. Add `extra` to the value blinding factor: `f’ = f’ + extra`.
		f.Add(&extra)

		// 3. Adjust the value commitment too: `V = V + extra·G`.
		extraG := multiplyBasePoint(extra)
		Vref := (*Point)(&V)
		Vref.add(&extraG)

		// 4. Note: as a result, the total blinding factor of the output will be equal to `q`.
	}

	// 14. [Encrypt Value](#encrypt-value): compute `(ev,ef)` from `(V’, value, f’, vek)`.
	ev := EncryptValue(V, amount, f, vek)

	// 15. [Create Asset Range Proof](#create-asset-range-proof): compute `ARP` from `(H’,(ea,ec),{H[i]},j,d)`.
	arp, err = CreateAssetRangeProof(Hprime, ea, H, j, d)
	if err != nil {
		return
	}

	// 16. [Create Value Range Proof](#create-value-range-proof): compute `VRP` from `(H’, V’, (ev,ef), N, value, {pt[i]}, f’, rek)`.
	vrp, err = CreateValueRangeProof(Hprime, V, ev, N, amount, ptchunks, f, rek)

	// 17. Create [encrypted asset ID descriptor](#encrypted-asset-id-descriptor) `AD` containing `H’` and `(ea,ec)`.
	AD = &BlindedAssetDescriptor{
		H:    Hprime,
		eaec: &ea,
	}

	// 18. Create [encrypted value descriptor](#encrypted-value-descriptor) `VD` containing `V’` and `(ev,ef)`.
	VD = &BlindedValueDescriptor{
		V:    V,
		evef: &ev,
	}

	// 19. Return `(AD, VD, ARP, VRP, c’, f’)`.
	return
}

// Inputs:
// 1. `rek`: the [record encryption key](#record-encryption-key).
// 2. `AD`: the [asset ID descriptor](#asset-id-descriptor).
// 3. `VD`: the [value descriptor](#value-descriptor).
// 4. `VRP`: the [value range proof](#value-range-proof) or an empty string.
//
// Outputs:
// 1. `assetID`: the output asset ID.
// 2. `value`: the output amount.
// 3. `c`: the output [cumulative blinding factor](#cumulative-blinding-factor) for the asset ID commitment `H`.
// 4. `f`: the output [value blinding factor](#value-blinding-factor).
// 5. `plaintext`: the binary string that has length of less than `32·(2·N-1)` bytes when encoded as [varstring31](data.md#varstring31).
func DecryptOutput(
	rek RecordKey,
	AD AssetDescriptor,
	VD ValueDescriptor,
	vrp *ValueRangeProof,
) (
	assetID AssetID,
	amount uint64,
	c Scalar,
	f Scalar,
	plaintext []byte,
	err error,
) {
	// 1. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)

	// 2. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
	vek := DeriveValueKey(iek)

	// 3. Decrypt asset ID:
	// 3.1. If `AD` is [nonblinded](#nonblinded-asset-id-descriptor): set `assetID` to the one stored in `AD`, set `c` to zero.
	switch bl := AD.(type) {
	case *NonblindedAssetDescriptor:
		assetID = bl.AssetID
		c = ZeroScalar
	case *BlindedAssetDescriptor:
		// 3.2. If `AD` is [blinded and not encrypted](#blinded-asset-id-descriptor), halt and return nil.
		eaec := bl.eaec
		if eaec == nil {
			err = ErrCannotDecrypt
			return
		}
		// 3.3. If `AD` is [encrypted](#encrypted-asset-id-descriptor), [Decrypt Asset ID](#decrypt-asset-id): compute `(assetID,c)` from `(H,(ea,ec),aek)`. If verification failed, halt and return `nil`.
		assetID, c, err = eaec.Decrypt(AD.Commitment(), aek)
		if err != nil {
			return
		}
	}

	// 4. Decrypt value:
	switch bl := VD.(type) {
	case *NonblindedValueDescriptor:
		// 4.1. If `VD` is [nonblinded](#nonblinded-value-descriptor): set `value` to the one stored in `VD`, set `f` to zero.
		amount = bl.Value
		f = ZeroScalar
	case *BlindedValueDescriptor:
		// 4.2. If `VD` is [blinded and not encrypted](#blinded-value-descriptor), halt and return nil.
		evef := bl.evef
		if evef == nil {
			err = ErrCannotDecrypt
			return
		}
		// 4.3. If `VD` is [encrypted](#encrypted-value-descriptor), [Decrypt Value](#decrypt-value): compute `(value, f)` from `(H,V,(ev,ef),vek)`. If verification failed, halt and return `nil`.
		var ok bool
		amount, f, ok = evef.Decrypt(VD.Commitment(), AD.Commitment(), vek)
		if !ok {
			err = ErrCannotDecrypt
			return
		}
	}

	// 5. If value range proof `VRP` is not empty:
	if vrp != nil {
		// 5.1. [Recover payload from Value Range Proof](#recover-payload-from-value-range-proof): compute a list of 32-byte chunks `{pt[i]}` from `(H,V,(ev,ef),VRP,value,f,rek)`. If verification failed, halt and return `nil`.
		var pt [][32]byte
		pt, err = vrp.RecoverPayload(AD.Commitment(), VD.Commitment(), VD.EncryptedValue(), amount, f, rek)
		if err != nil {
			return
		}

		// 5.2. Flatten the array `{pt[i]}` in a binary string and decode it using [varstring31](data.md#varstring31) encoding. If decoding fails, halt and return `nil`.
		ptbuf := bufpool.Get()
		defer bufpool.Put(ptbuf)
		for _, p := range pt {
			ptbuf.Write(p[:])
		}
		plaintext, _, err = blockchain.ReadVarstr31(bytes.NewReader(ptbuf.Bytes()))

		// 6. If value range proof `VRP` is empty, set `plaintext` to an empty string.
	} else {
		plaintext = []byte{}
	}

	return
}
