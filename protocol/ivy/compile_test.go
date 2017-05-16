package ivy

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"chain/protocol/ivy/ivytest"
	"chain/protocol/vm"
	"chain/testutil"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name     string
		contract string
		want     CompileResult
	}{
		{
			"TrivialLock",
			ivytest.TrivialLock,
			CompileResult{
				Name:    "TrivialLock",
				Program: mustDecodeHex("51"),
				Value:   "locked",
				Clauses: []ClauseInfo{{
					Name: "trivialUnlock",
					Values: []ValueInfo{{
						Name: "locked",
					}},
				}},
			},
		},
		{
			"LockWithPublicKey",
			ivytest.LockWithPublicKey,
			CompileResult{
				Name:    "LockWithPublicKey",
				Program: mustDecodeHex("ae7cac"),
				Value:   "locked",
				Params: []ContractParam{{
					Name: "publicKey",
					Typ:  "PublicKey",
				}},
				Clauses: []ClauseInfo{{
					Name: "unlockWithSig",
					Args: []ClauseArg{{
						Name: "sig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name: "locked",
					}},
				}},
			},
		},
		{
			"LockWithPublicKeyHash",
			ivytest.LockWithPKHash,
			CompileResult{
				Name:    "LockWithPublicKeyHash",
				Program: mustDecodeHex("5279aa887cae7cac"),
				Value:   "value",
				Params: []ContractParam{{
					Name: "pubKeyHash",
					Typ:  "Sha3(PublicKey)",
				}},
				Clauses: []ClauseInfo{{
					Name: "spend",
					Args: []ClauseArg{{
						Name: "pubKey",
						Typ:  "PublicKey",
					}, {
						Name: "sig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name: "value",
					}},
					HashCalls: []HashCall{{
						HashType: "sha3",
						Arg:      "pubKey",
						ArgType:  "PublicKey",
					}},
				}},
			},
		},
		{
			"LockWith2of3Keys",
			ivytest.LockWith2of3Keys,
			CompileResult{
				Name:    "LockWith3Keys",
				Program: mustDecodeHex("537a547a526bae71557a536c7cad"),
				Value:   "locked",
				Params: []ContractParam{{
					Name: "pubkey1",
					Typ:  "PublicKey",
				}, {
					Name: "pubkey2",
					Typ:  "PublicKey",
				}, {
					Name: "pubkey3",
					Typ:  "PublicKey",
				}},
				Clauses: []ClauseInfo{{
					Name: "unlockWith2Sigs",
					Args: []ClauseArg{{
						Name: "sig1",
						Typ:  "Signature",
					}, {
						Name: "sig2",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name: "locked",
					}},
				}},
			},
		},
		{
			"LockToOutput",
			ivytest.LockToOutput,
			CompileResult{
				Name:    "LockToOutput",
				Program: mustDecodeHex("0000c3c251557ac1"),
				Value:   "locked",
				Params: []ContractParam{{
					Name: "address",
					Typ:  "Program",
				}},
				Clauses: []ClauseInfo{{
					Name: "relock",
					Values: []ValueInfo{{
						Name:    "locked",
						Program: "address",
					}},
				}},
			},
		},
		{
			"TradeOffer",
			ivytest.TradeOffer,
			CompileResult{
				Name:    "TradeOffer",
				Program: mustDecodeHex("547a641300000000007251557ac16323000000547a547aae7cac690000c3c251577ac1"),
				Value:   "offered",
				Params: []ContractParam{{
					Name: "requestedAsset",
					Typ:  "Asset",
				}, {
					Name: "requestedAmount",
					Typ:  "Amount",
				}, {
					Name: "sellerProgram",
					Typ:  "Program",
				}, {
					Name: "sellerKey",
					Typ:  "PublicKey",
				}},
				Clauses: []ClauseInfo{{
					Name: "trade",
					Values: []ValueInfo{{
						Name:    "payment",
						Program: "sellerProgram",
						Asset:   "requestedAsset",
						Amount:  "requestedAmount",
					}, {
						Name: "offered",
					}},
				}, {
					Name: "cancel",
					Args: []ClauseArg{{
						Name: "sellerSig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name:    "offered",
						Program: "sellerProgram",
					}},
				}},
			},
		},
		{
			"EscrowedTransfer",
			ivytest.EscrowedTransfer,
			CompileResult{
				Name:    "EscrowedTransfer",
				Program: mustDecodeHex("537a641b000000537a7cae7cac690000c3c251567ac1632a000000537a7cae7cac690000c3c251557ac1"),
				Value:   "value",
				Params: []ContractParam{{
					Name: "agent",
					Typ:  "PublicKey",
				}, {
					Name: "sender",
					Typ:  "Program",
				}, {
					Name: "recipient",
					Typ:  "Program",
				}},
				Clauses: []ClauseInfo{{
					Name: "approve",
					Args: []ClauseArg{{
						Name: "sig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name:    "value",
						Program: "recipient",
					}},
				}, {
					Name: "reject",
					Args: []ClauseArg{{
						Name: "sig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name:    "value",
						Program: "sender",
					}},
				}},
			},
		},
		{
			"CollateralizedLoan",
			ivytest.CollateralizedLoan,
			CompileResult{
				Name:    "CollateralizedLoan",
				Program: mustDecodeHex("557a641c00000000007251567ac1695100c3c251567ac163280000007bc59f690000c3c251577ac1"),
				Value:   "collateral",
				Params: []ContractParam{{
					Name: "balanceAsset",
					Typ:  "Asset",
				}, {
					Name: "balanceAmount",
					Typ:  "Amount",
				}, {
					Name: "deadline",
					Typ:  "Time",
				}, {
					Name: "lender",
					Typ:  "Program",
				}, {
					Name: "borrower",
					Typ:  "Program",
				}},
				Clauses: []ClauseInfo{{
					Name: "repay",
					Values: []ValueInfo{
						{
							Name:    "payment",
							Program: "lender",
							Asset:   "balanceAsset",
							Amount:  "balanceAmount",
						},
						{
							Name:    "collateral",
							Program: "borrower",
						},
					},
				}, {
					Name: "default",
					Values: []ValueInfo{
						{
							Name:    "collateral",
							Program: "lender",
						},
					},
					Mintimes: []string{"deadline"},
				}},
			},
		},
		{
			"RevealPreimage",
			ivytest.RevealPreimage,
			CompileResult{
				Name:    "RevealPreimage",
				Program: mustDecodeHex("7caa87"),
				Value:   "value",
				Params: []ContractParam{{
					Name: "hash",
					Typ:  "Sha3(String)",
				}},
				Clauses: []ClauseInfo{{
					Name: "reveal",
					Args: []ClauseArg{{
						Name: "string",
						Typ:  "String",
					}},
					Values: []ValueInfo{{
						Name: "value",
					}},
					HashCalls: []HashCall{{
						HashType: "sha3",
						Arg:      "string",
						ArgType:  "String",
					}},
				}},
			},
		},
		{
			"CallOptionWithSettlement",
			ivytest.CallOptionWithSettlement,
			CompileResult{
				Name:    "CallOptionWithSettlement",
				Program: mustDecodeHex("567a76529c64390000006427000000557ac6a06971ae7cac6900007b537a51557ac16349000000557ac59f690000c3c251577ac1634900000075577a547aae7cac69557a547aae7cac"),
				Value:   "underlying",
				Params: []ContractParam{{
					Name: "strikePrice",
					Typ:  "Amount",
				}, {
					Name: "strikeCurrency",
					Typ:  "Asset",
				}, {
					Name: "sellerProgram",
					Typ:  "Program",
				}, {
					Name: "sellerKey",
					Typ:  "PublicKey",
				}, {
					Name: "buyerKey",
					Typ:  "PublicKey",
				}, {
					Name: "deadline",
					Typ:  "Time",
				}},
				Clauses: []ClauseInfo{{
					Name: "exercise",
					Args: []ClauseArg{{
						Name: "buyerSig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name:    "payment",
						Program: "sellerProgram",
						Asset:   "strikeCurrency",
						Amount:  "strikePrice",
					}, {
						Name: "underlying",
					}},
					Maxtimes: []string{"deadline"},
				}, {
					Name: "expire",
					Values: []ValueInfo{{
						Name:    "underlying",
						Program: "sellerProgram",
					}},
					Mintimes: []string{"deadline"},
				}, {
					Name: "settle",
					Args: []ClauseArg{{
						Name: "sellerSig",
						Typ:  "Signature",
					}, {
						Name: "buyerSig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name: "underlying",
					}},
				}},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.contract)
			got, err := Compile(r, nil)
			if err != nil {
				t.Fatal(err)
			}
			labels := got.Labels
			got.Labels = nil // to make DeepEqual easier
			gotProg, _ := vm.Disassemble(got.Program, labels)
			if !testutil.DeepEqual(got, c.want) {
				wantProg, _ := vm.Disassemble(c.want.Program, labels)
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(c.want)
				t.Errorf("got %s [prog: %s]\nwant %s [prog: %s]", string(gotJSON), gotProg, wantJSON, wantProg)
			} else {
				t.Log(gotProg)
			}
		})
	}
}

func mustDecodeHex(h string) []byte {
	bits, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return bits
}
