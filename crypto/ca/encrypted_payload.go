package ca

func EncryptPayload(
	plaintext [][32]byte,
	ek [32]byte,
) [][32]byte {
	n := len(plaintext)
	ciphertext := make([][32]byte, n+1)

	keyer := shake256(ek[:])
	mac := hasher256(ek[:])
	for i := 0; i < n; i++ {
		// 1. Calculate a keystream, a sequence of 32-byte random values: `{keystream[i]} = SHAKE256(ek, 8·(32·n))`.
		keyer.Read(ciphertext[i][:])

		// 2. Encrypt the plaintext payload: `{ct[i]} = {pt[i] XOR keystream[i]}`.
		ciphertext[i] = xor256(plaintext[i][:], ciphertext[i][:])

		// 3. Calculate MAC: `mac = SHA3-256(ek || ct[0] || ... || ct[n-1])`.
		mac.Write(ciphertext[i][:])
	}
	// 4. Return a sequence of `n+1` 32-byte elements: `{ct[0], ..., ct[n-1], mac}`.
	mac.Sum(ciphertext[n][:0])
	return ciphertext
}

func DecryptPayload(
	ciphertext [][32]byte,
	ek [32]byte,
) (plaintext [][32]byte, ok bool) {
	n := len(ciphertext)
	if n < 1 {
		return [][32]byte{}, false
	}
	n--
	plaintext = make([][32]byte, n)

	keyer := shake256(ek[:])
	mac := hasher256(ek[:])
	for i := 0; i < n; i++ {
		// 1. Calculate MAC’: `mac’ = SHA3-256(ek || ct[0] || ... || ct[n-1])`.
		mac.Write(ciphertext[i][:])

		// 4. Calculate a keystream, a sequence of 32-byte random values: `{keystream[i]} = SHAKE256(ek, 8·(32·n))`.
		keyer.Read(plaintext[i][:])

		// 5. Decrypt the plaintext payload: `{pt[i]} = {ct[i] XOR keystream[i]}`.
		plaintext[i] = xor256(ciphertext[i][:], plaintext[i][:])
	}
	// 2. Extract the transmitted MAC: `mac = ct[n]`.
	receivedMAC := [32]byte{}
	mac.Sum(receivedMAC[:0])

	// 3. Compare calculated  `mac’` with the received `mac`. If they are not equal, return `nil`.
	if !constTimeEqual(receivedMAC[:], ciphertext[n][:]) {
		return [][32]byte{}, false
	}

	// 6. Return `{pt[i]}`.
	return plaintext, true
}
