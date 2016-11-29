package ca

func CreateTransientIssuanceKey(
	assetid AssetID,
	aek AssetKey,
) (y Scalar, Y Point) {

	// 1. Calculate `secret = SHA3-512(0xa1 || assetid || aek)`.
	// 2. Calculate scalar `y` by reducing the `secret` modulo subgroup order `L`: `y = secret mod L`.
	y = reducedScalar(hash512([]byte{0xa1}, assetid[:], aek[:]))

	// 3. Calculate point `Y` by multiplying base point by `y`: `Y = y·G`.
	Y = multiplyBasePoint(y)

	// 4. Return key pair `(y,Y)`.
	return y, Y
}
