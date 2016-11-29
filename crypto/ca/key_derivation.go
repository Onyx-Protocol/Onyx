package ca

type (
	RecordKey       [32]byte
	IntermediateKey [32]byte
	AssetKey        [32]byte
	ValueKey        [32]byte
)

// Intermediate encryption key (IEK) allows
// decrypting the asset ID and the value in the output commitment.
// It is derived from the REK:
//     iek = SHA3-256(0x00 || rek)
func DeriveIntermediateKey(rek RecordKey) IntermediateKey {
	return hash256([]byte{0x00}, rek[:])
}

// Asset ID encryption key (AEK) allows decrypting the asset ID
// in the output commitment. It is derived from the IEK as follows:
//     aek = SHA3-256(0x00 || iek)
func DeriveAssetKey(iek IntermediateKey) AssetKey {
	return hash256([]byte{0x00}, iek[:])
}

// Value encryption key (VEK) allows decrypting the amount
// in the output commitment. It is derived from the IEK as follows:
//     vek = SHA3-256(0x01 || iek)
func DeriveValueKey(iek IntermediateKey) ValueKey {
	return hash256([]byte{0x01}, iek[:])
}
