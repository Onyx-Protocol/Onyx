package ca

import "testing"

func TestEncryptAssetID(t *testing.T) {
	wantea := fromHex256("ae58d12da3fd13756a440b5adb76770ac3a18407c6d01097ea6841745f2829aa")
	wantec := fromHex256("da6706481aaa573170e6835f0538e9c39346a421e59be421f05df77e80e4fc42")
	got := EncryptAssetID(
		AssetID{},
		AssetCommitment(mustDecodePoint(fromHex256("94e7bc9d0bf5faf65ecf09e3f6c9d736cea50696163b3a30eb1ff5c4d042437a"))),
		Scalar(fromHex256("4f1b5c5a3689e4d8b53d10849fc29868edc81d6dd306299ebe95860f4eb1600a")),
		AssetKey(fromHex256("e8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62")),
	)
	if got.AssetID != wantea {
		t.Errorf("Encrypted asset ID is not computed correctly:\ngot:  %x\nwant: %x", got.AssetID[:], wantea[:])
	}
	if got.BlindingFactor != wantec {
		t.Errorf("Encrypted cumulative blinding factor is not computed correctly:\ngot:  %x\nwant: %x", got.BlindingFactor[:], wantec[:])
	}
}

func TestDecryptAssetID(t *testing.T) {
	wanta := AssetID{}
	wantc := fromHex256("4f1b5c5a3689e4d8b53d10849fc29868edc81d6dd306299ebe95860f4eb1600a")

	enc := EncryptedAssetID{
		AssetID:        fromHex256("ae58d12da3fd13756a440b5adb76770ac3a18407c6d01097ea6841745f2829aa"),
		BlindingFactor: fromHex256("da6706481aaa573170e6835f0538e9c39346a421e59be421f05df77e80e4fc42"),
	}

	ac := AssetCommitment(mustDecodePoint(fromHex256("94e7bc9d0bf5faf65ecf09e3f6c9d736cea50696163b3a30eb1ff5c4d042437a")))
	bf := AssetKey(fromHex256("e8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62"))

	gota, gotc, err := enc.Decrypt(ac, bf)
	if err != nil {
		t.Fatal(err)
	}
	if gota != wanta {
		t.Errorf("Got A %x, want %x", gota[:], wanta[:])
	}
	if gotc != wantc {
		t.Errorf("Got C %x, want %x", gotc[:], wantc[:])
	}

	ac2 := AssetCommitment(multiplyPoint(cofactor, Point(ac)))
	_, _, err = enc.Decrypt(ac2, bf)
	if err == nil {
		t.Error("unexpected success decrypting with alternate asset commitment")
	}

	bf[0] ^= 1
	_, _, err = enc.Decrypt(ac, bf)
	if err == nil {
		t.Error("unexpected success decrypting with alternate asset key")
	}
}

func TestFailToDecryptAssetID(t *testing.T) {
	enc := EncryptedAssetID{
		AssetID:        fromHex256("ae58d12da3fd13756a440b5adb76770ac3a18407c6d01097ea6841745f2829aa"),
		BlindingFactor: fromHex256("da6706481aaa573170e6835f0538e9c39346a421e59be421f05df77e80e4fc42"),
	}
	_, _, err := enc.Decrypt(
		AssetCommitment(mustDecodePoint(fromHex256("94e7bc9d0bf5faf65ecf09e3f6c9d736cea50696163b3a30eb1ff5c4d042437a"))),
		AssetKey(fromHex256("f8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62")),
	)
	if err == nil {
		t.Errorf("Did not get an error when decrypting asset ID with incorrect key")
	}

	enc = EncryptedAssetID{
		AssetID:        fromHex256("ae58d12da3fd13756a440b5adb76770ac3a18407c6d01097ea6841745f2829aa"),
		BlindingFactor: fromHex256("1c1b0e054fee6ce657a6f1e79429d5ade93cf281ca702844d404b200bb83669f"),
	}
	_, _, err = enc.Decrypt(
		AssetCommitment(mustDecodePoint(fromHex256("94e7bc9d0bf5faf65ecf09e3f6c9d736cea50696163b3a30eb1ff5c4d042437a"))),
		AssetKey(fromHex256("e8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62")),
	)
	if err == nil {
		t.Errorf("Did not get an error when decrypting asset ID with incorrect encrypted cumulative blinding factor")
	}

	enc = EncryptedAssetID{
		AssetID:        fromHex256("15a2f2f785e1734d6b3c3fe54461fae8e5d0e09de83d02cc83013f4405be2805"),
		BlindingFactor: fromHex256("da6706481aaa573170e6835f0538e9c39346a421e59be421f05df77e80e4fc42"),
	}
	_, _, err = enc.Decrypt(
		AssetCommitment(mustDecodePoint(fromHex256("94e7bc9d0bf5faf65ecf09e3f6c9d736cea50696163b3a30eb1ff5c4d042437a"))),
		AssetKey(fromHex256("e8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62")),
	)
	if err == nil {
		t.Errorf("Did not get an error when decrypting asset ID with incorrect encrypted asset ID")
	}

	enc = EncryptedAssetID{
		AssetID:        fromHex256("ae58d12da3fd13756a440b5adb76770ac3a18407c6d01097ea6841745f2829aa"),
		BlindingFactor: fromHex256("da6706481aaa573170e6835f0538e9c39346a421e59be421f05df77e80e4fc42"),
	}
	_, _, err = enc.Decrypt(
		AssetCommitment(mustDecodePoint(fromHex256("24ba76fc2012532877e7b5048f68c616d044d3a17cf501f336f40743dd9dbdb4"))),
		AssetKey(fromHex256("e8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62")),
	)
	if err == nil {
		t.Errorf("Did not get an error when decrypting asset ID with incorrect asset commitment")
	}
}
