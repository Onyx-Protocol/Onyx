// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"chain/cos/txscript"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

// decodeHex decodes the passed hex string and returns the resulting bytes.  It
// panics if an error occurs.  This is only used in the tests as a helper since
// the only way it can fail is if there is an error in the test source code.
func decodeHex(hexStr string) []byte {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		panic("invalid hex string in test source: err " + err.Error() +
			", hex: " + hexStr)
	}

	return b
}

// mustParseScriptString parses the passed short form script and returns the
// resulting bytes.  It panics if an error occurs.  This is only used in the
// tests as a helper since the only way it can fail is if there is an error in
// the test source code.
func mustParseScriptString(script string) []byte {
	s, err := txscript.ParseScriptString(script)
	if err != nil {
		panic("invalid short form script in test source: err " +
			err.Error() + ", script: " + script)
	}

	return s
}

// bogusAddress implements the btcutil.Address interface so the tests can ensure
// unsupported address types are handled properly.
type bogusAddress struct{}

// EncodeAddress simply returns an empty string.  It exists to satsify the
// btcutil.Address interface.
func (b *bogusAddress) EncodeAddress() string {
	return ""
}

// ScriptAddress simply returns an empty byte slice.  It exists to satsify the
// btcutil.Address interface.
func (b *bogusAddress) ScriptAddress() []byte {
	return nil
}

// IsForNet lies blatantly to satisfy the btcutil.Address interface.
func (b *bogusAddress) IsForNet(chainParams *chaincfg.Params) bool {
	return true // why not?
}

// String simply returns an empty string.  It exists to satsify the
// btcutil.Address interface.
func (b *bogusAddress) String() string {
	return ""
}

// TestPayToAddrScript ensures the PayToAddrScript function generates the
// correct scripts for the various types of addresses.
func TestPayToAddrScript(t *testing.T) {
	t.Parallel()

	// 1MirQ9bwyQcGVJPwKUgapu5ouK2E2Ey4gX
	p2pkhMain, err := btcutil.NewAddressPubKeyHash(decodeHex("e34cce70c863"+
		"73273efcc54ce7d2a491bb4a0e84"), &chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create public key hash address: %v", err)
		return
	}

	// Taken from transaction:
	// b0539a45de13b3e0403909b8bd1a555b8cbe45fd4e3f3fda76f3a5f52835c29d
	p2shMain, _ := btcutil.NewAddressScriptHashFromHash(decodeHex("e8c300"+
		"c87986efa84c37c0519929019ef86eb5b4"), &chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create script hash address: %v", err)
		return
	}

	//  mainnet p2pk 13CG6SJ3yHUXo4Cr2RY4THLLJrNFuG3gUg
	p2pkCompressedMain, err := btcutil.NewAddressPubKey(decodeHex("02192d74"+
		"d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create pubkey address (compressed): %v",
			err)
		return
	}
	p2pkCompressed2Main, err := btcutil.NewAddressPubKey(decodeHex("03b0bd"+
		"634234abbb1ba1e986e884185c61cf43e001f9137f23c2c409273eb16e65"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create pubkey address (compressed 2): %v",
			err)
		return
	}

	p2pkUncompressedMain, err := btcutil.NewAddressPubKey(decodeHex("0411db"+
		"93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5cb2"+
		"e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b412a3"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create pubkey address (uncompressed): %v",
			err)
		return
	}

	contractHash := decodeHex("5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456")
	p2cParams := make([][]byte, 0)

	v1 := []byte{0x01}

	// zero-param p2c addr
	p2c0 := txscript.NewAddressContractHash(contractHash, v1, p2cParams)

	// one-param p2c addr
	p2cParams = append(p2cParams, decodeHex("61"))
	p2c1 := txscript.NewAddressContractHash(contractHash, v1, p2cParams)

	// two-param p2c addr
	p2cParams = append(p2cParams, decodeHex("62"))
	p2c2 := txscript.NewAddressContractHash(contractHash, v1, p2cParams)

	tests := []struct {
		in       btcutil.Address
		expected string
		err      error
	}{
		// pay-to-contract, zero parameters
		{
			p2c0,
			"1 DROP DUP HASH256 DATA_32 0x5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456 EQUALVERIFY EVAL",
			nil,
		},

		// pay-to-contract, one parameter
		{
			p2c1,
			"1 DROP DATA_1 0x61 1 ROLL DUP HASH256 DATA_32 0x5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456 EQUALVERIFY EVAL",
			nil,
		},

		// pay-to-contract, two parameters
		{
			p2c2,
			"1 DROP DATA_1 0x62 DATA_1 0x61 2 ROLL DUP HASH256 DATA_32 0x5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456 EQUALVERIFY EVAL",
			nil,
		},

		// pay-to-pubkey-hash address on mainnet
		{
			p2pkhMain,
			"DUP HASH160 DATA_20 0xe34cce70c86373273efcc54ce7d2a4" +
				"91bb4a0e8488 CHECKSIG",
			nil,
		},
		// pay-to-script-hash address on mainnet
		{
			p2shMain,
			"HASH160 DATA_20 0xe8c300c87986efa84c37c0519929019ef8" +
				"6eb5b4 EQUAL",
			nil,
		},
		// pay-to-pubkey address on mainnet. compressed key.
		{
			p2pkCompressedMain,
			"DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c3" +
				"ebec3a957724895dca52c6b4 CHECKSIG",
			nil,
		},
		// pay-to-pubkey address on mainnet. compressed key (other way).
		{
			p2pkCompressed2Main,
			"DATA_33 0x03b0bd634234abbb1ba1e986e884185c61cf43e001" +
				"f9137f23c2c409273eb16e65 CHECKSIG",
			nil,
		},
		// pay-to-pubkey address on mainnet. uncompressed key.
		{
			p2pkUncompressedMain,
			"DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382e" +
				"97b1482ecad7b148a6909a5cb2e0eaddfb84ccf97444" +
				"64f82e160bfa9b8b64f9d4c03f999b8643f656b412a3 " +
				"CHECKSIG",
			nil,
		},

		// Supported address types with nil pointers.
		{(*btcutil.AddressPubKeyHash)(nil), "", txscript.ErrUnsupportedAddress},
		{(*btcutil.AddressScriptHash)(nil), "", txscript.ErrUnsupportedAddress},
		{(*btcutil.AddressPubKey)(nil), "", txscript.ErrUnsupportedAddress},

		// Unsupported address type.
		{&bogusAddress{}, "", txscript.ErrUnsupportedAddress},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		pkScript, err := txscript.PayToAddrScript(test.in)
		if err != test.err {
			t.Errorf("PayToAddrScript #%d unexpected error - "+
				"got %v, want %v", i, err, test.err)
			continue
		}

		expected := mustParseScriptString(test.expected)
		if !bytes.Equal(pkScript, expected) {
			pkScriptStr, _ := txscript.DisasmString(pkScript)
			expectedStr, _ := txscript.DisasmString(expected)

			t.Errorf("PayToAddrScript #%d, test.in is %+v\ngot: %s (%x)\nwant: %s (%x)",
				i, test.in, pkScriptStr, pkScript, expectedStr, expected)
			continue
		}
	}
}

// TestMultiSigScript ensures the MultiSigScript function returns the expected
// scripts and errors.
func TestMultiSigScript(t *testing.T) {
	t.Parallel()

	//  mainnet p2pk 13CG6SJ3yHUXo4Cr2RY4THLLJrNFuG3gUg
	p2pkCompressedMain, err := btcutil.NewAddressPubKey(decodeHex("02192d7"+
		"4d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create pubkey address (compressed): %v",
			err)
		return
	}
	p2pkCompressed2Main, err := btcutil.NewAddressPubKey(decodeHex("03b0bd"+
		"634234abbb1ba1e986e884185c61cf43e001f9137f23c2c409273eb16e65"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create pubkey address (compressed 2): %v",
			err)
		return
	}

	p2pkUncompressedMain, err := btcutil.NewAddressPubKey(decodeHex("0411d"+
		"b93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5c"+
		"b2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b41"+
		"2a3"), &chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Unable to create pubkey address (uncompressed): %v",
			err)
		return
	}

	tests := []struct {
		keys      []*btcutil.AddressPubKey
		nrequired int
		expected  string
		err       error
	}{
		{
			[]*btcutil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			1,
			"1 DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c" +
				"3ebec3a957724895dca52c6b4 DATA_33 0x03b0bd634" +
				"234abbb1ba1e986e884185c61cf43e001f9137f23c2c4" +
				"09273eb16e65 2 CHECKMULTISIG",
			nil,
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			2,
			"2 DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c" +
				"3ebec3a957724895dca52c6b4 DATA_33 0x03b0bd634" +
				"234abbb1ba1e986e884185c61cf43e001f9137f23c2c4" +
				"09273eb16e65 2 CHECKMULTISIG",
			nil,
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			3,
			"",
			txscript.ErrBadNumRequired,
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkUncompressedMain,
			},
			1,
			"1 DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382" +
				"e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf97444" +
				"64f82e160bfa9b8b64f9d4c03f999b8643f656b412a3 " +
				"1 CHECKMULTISIG",
			nil,
		},
		{
			[]*btcutil.AddressPubKey{
				p2pkUncompressedMain,
			},
			2,
			"",
			txscript.ErrBadNumRequired,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		script, err := txscript.MultiSigScript(test.keys,
			test.nrequired)
		if err != test.err {
			t.Errorf("MultiSigScript #%d unexpected error - "+
				"got %v, want %v", i, err, test.err)
			continue
		}

		expected := mustParseScriptString(test.expected)
		if !bytes.Equal(script, expected) {
			t.Errorf("MultiSigScript #%d got: %x\nwant: %x",
				i, script, expected)
			continue
		}
	}
}

// TestCalcMultiSigStats ensures the CalcMutliSigStats function returns the
// expected errors.
func TestCalcMultiSigStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		script string
		err    error
	}{
		{
			name: "short script",
			script: "0x046708afdb0fe5548271967f1a67130b7105cd6a828" +
				"e03909a67962e0ea1f61d",
			err: txscript.ErrStackShortScript,
		},
		{
			name: "stack underflow",
			script: "RETURN DATA_41 0x046708afdb0fe5548271967f1a" +
				"67130b7105cd6a828e03909a67962e0ea1f61deb649f6" +
				"bc3f4cef308",
			err: txscript.ErrStackUnderflow,
		},
		{
			name: "multisig script",
			script: "0 DATA_72 0x30450220106a3e4ef0b51b764a2887226" +
				"2ffef55846514dacbdcbbdd652c849d395b4384022100" +
				"e03ae554c3cbb40600d31dd46fc33f25e47bf8525b1fe" +
				"07282e3b6ecb5f3bb2801 CODESEPARATOR 1 DATA_33 " +
				"0x0232abdc893e7f0631364d7fd01cb33d24da45329a0" +
				"0357b3a7886211ab414d55a 1 CHECKMULTISIG",
			err: nil,
		},
	}

	for i, test := range tests {
		script := mustParseScriptString(test.script)
		if _, _, err := txscript.CalcMultiSigStats(script); err != test.err {
			t.Errorf("CalcMultiSigStats #%d (%s) unexpected "+
				"error\ngot: %v\nwant: %v", i, test.name, err,
				test.err)
		}
	}
}

// scriptClassTest houses a test used to ensure various scripts have the
// expected class.
type scriptClassTest struct {
	name   string
	script string
	class  txscript.ScriptClass
}

// scriptClassTests houses several test scripts used to ensure various class
// determination is working as expected.  It's defined as a test global versus
// inside a function scope since this spans both the standard tests and the
// consensus tests (pay-to-script-hash is part of consensus).
var scriptClassTests = []scriptClassTest{
	{
		name: "Pay Pubkey",
		script: "DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382e" +
			"97b1482ecad7b148a6909a5cb2e0eaddfb84ccf9744464f82e16" +
			"0bfa9b8b64f9d4c03f999b8643f656b412a3 CHECKSIG",
		class: txscript.PubKeyTy,
	},
	// tx 599e47a8114fe098103663029548811d2651991b62397e057f0c863c2bc9f9ea
	{
		name: "Pay PubkeyHash",
		script: "DUP HASH160 DATA_20 0x660d4ef3a743e3e696ad990364e555" +
			"c271ad504b EQUALVERIFY CHECKSIG",
		class: txscript.PubKeyHashTy,
	},
	// part of tx 6d36bc17e947ce00bb6f12f8e7a56a1585c5a36188ffa2b05e10b4743273a74b
	// codeseparator parts have been elided. (bitcoin core's checks for
	// multisig type doesn't have codesep either).
	{
		name: "multisig",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4" +
			"5329a00357b3a7886211ab414d55a 1 CHECKMULTISIG",
		class: txscript.MultiSigTy,
	},
	// tx e5779b9e78f9650debc2893fd9636d827b26b4ddfa6a8172fe8708c924f5c39d
	{
		name: "P2SH",
		script: "HASH160 DATA_20 0x433ec2ac1ffa1b7b7d027f564529c57197f" +
			"9ae88 EQUAL",
		class: txscript.ScriptHashTy,
	},
	{
		// Nulldata with no data at all.
		name:   "nulldata",
		script: "RETURN",
		class:  txscript.NullDataTy,
	},
	{
		// Nulldata with small data.
		name:   "nulldata2",
		script: "RETURN DATA_8 0x046708afdb0fe554",
		class:  txscript.NullDataTy,
	},
	{
		// Nulldata with max allowed data.
		name: "nulldata3",
		script: "RETURN PUSHDATA1 0x50 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a828e03909a67" +
			"962e0ea1f61deb649f6bc3f4cef3",
		class: txscript.NullDataTy,
	},
	{
		// Nulldata with more than max allowed data (so therefore
		// nonstandard)
		name: "nulldata4",
		script: "RETURN PUSHDATA1 0x51 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a828e03909a67" +
			"962e0ea1f61deb649f6bc3f4cef308",
		class: txscript.NonStandardTy,
	},
	{
		// Almost nulldata, but add an additional opcode after the data
		// to make it nonstandard.
		name:   "nulldata5",
		script: "RETURN 4 TRUE",
		class:  txscript.NonStandardTy,
	},

	// The next few are almost multisig (it is the more complex script type)
	// but with various changes to make it fail.
	{
		// Multisig but invalid nsigs.
		name: "strange 1",
		script: "DUP DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da45" +
			"329a00357b3a7886211ab414d55a 1 CHECKMULTISIG",
		class: txscript.NonStandardTy,
	},
	{
		// Multisig but invalid pubkey.
		name:   "strange 2",
		script: "1 1 1 CHECKMULTISIG",
		class:  txscript.NonStandardTy,
	},
	{
		// Multisig but no matching npubkeys opcode.
		name: "strange 3",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4532" +
			"9a00357b3a7886211ab414d55a DATA_33 0x0232abdc893e7f0" +
			"631364d7fd01cb33d24da45329a00357b3a7886211ab414d55a " +
			"CHECKMULTISIG",
		class: txscript.NonStandardTy,
	},
	{
		// Multisig but with multisigverify.
		name: "strange 4",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4532" +
			"9a00357b3a7886211ab414d55a 1 CHECKMULTISIGVERIFY",
		class: txscript.NonStandardTy,
	},
	{
		// Multisig but wrong length.
		name:   "strange 5",
		script: "1 CHECKMULTISIG",
		class:  txscript.NonStandardTy,
	},
	{
		name:   "doesn't parse",
		script: "DATA_5 0x01020304",
		class:  txscript.NonStandardTy,
	},
}

// TestScriptClass ensures all the scripts in scriptClassTests have the expected
// class.
func TestScriptClass(t *testing.T) {
	t.Parallel()

	for _, test := range scriptClassTests {
		script := mustParseScriptString(test.script)
		class := txscript.GetScriptClass(script)
		if class != test.class {
			t.Errorf("%s: expected %s got %s", test.name,
				test.class, class)
			return
		}
	}
}

// TestStringifyClass ensures the script class string returns the expected
// string for each script class.
func TestStringifyClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		class    txscript.ScriptClass
		stringed string
	}{
		{
			name:     "nonstandardty",
			class:    txscript.NonStandardTy,
			stringed: "nonstandard",
		},
		{
			name:     "pubkey",
			class:    txscript.PubKeyTy,
			stringed: "pubkey",
		},
		{
			name:     "pubkeyhash",
			class:    txscript.PubKeyHashTy,
			stringed: "pubkeyhash",
		},
		{
			name:     "scripthash",
			class:    txscript.ScriptHashTy,
			stringed: "scripthash",
		},
		{
			name:     "multisigty",
			class:    txscript.MultiSigTy,
			stringed: "multisig",
		},
		{
			name:     "nulldataty",
			class:    txscript.NullDataTy,
			stringed: "nulldata",
		},
		{
			name:     "broken",
			class:    txscript.ScriptClass(255),
			stringed: "Invalid",
		},
	}

	for _, test := range tests {
		typeString := test.class.String()
		if typeString != test.stringed {
			t.Errorf("%s: got %#q, want %#q", test.name,
				typeString, test.stringed)
		}
	}
}
