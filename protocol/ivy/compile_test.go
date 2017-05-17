package ivy

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"chain/protocol/vm"
	"chain/testutil"
)

const trivialLock = `
contract TrivialLock() locks locked {
  clause trivialUnlock() {
    unlock locked
  }
}
`

const lockWithPublicKey = `
contract LockWithPublicKey(publicKey: PublicKey) locks locked {
  clause unlockWithSig(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    unlock locked
  }
}
`

const lockWithPKHash = `
contract LockWithPublicKeyHash(pubKeyHash: Hash) locks value {
  clause spend(pubKey: PublicKey, sig: Signature) {
    verify sha3(pubKey) == pubKeyHash
    verify checkTxSig(pubKey, sig)
    unlock value
  }
}
`

const lockWith2of3Keys = `
contract LockWith3Keys(pubkey1, pubkey2, pubkey3: PublicKey) locks locked {
  clause unlockWith2Sigs(sig1, sig2: Signature) {
    verify checkTxMultiSig([pubkey1, pubkey2, pubkey3], [sig1, sig2])
    unlock locked
  }
}
`

const lockToOutput = `
contract LockToOutput(address: Program) locks locked {
  clause relock() {
    lock locked with address
  }
}
`

const tradeOffer = `
contract TradeOffer(requestedAsset: Asset, requestedAmount: Amount, sellerProgram: Program, sellerKey: PublicKey) locks offered {
  clause trade() requires payment: requestedAmount of requestedAsset {
    lock payment with sellerProgram
    unlock offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    lock offered with sellerProgram
  }
}
`

const escrowedTransfer = `
contract EscrowedTransfer(agent: PublicKey, sender: Program, recipient: Program) locks value {
  clause approve(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock value with recipient
  }
  clause reject(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock value with sender
  }
}
`

const collateralizedLoan = `
contract CollateralizedLoan(balanceAsset: Asset, balanceAmount: Amount, deadline: Time, lender: Program, borrower: Program) locks collateral {
  clause repay() requires payment: balanceAmount of balanceAsset {
    lock payment with lender
    lock collateral with borrower
  }
  clause default() {
    verify after(deadline)
    lock collateral with lender
  }
}
`

const revealPreimage = `
contract RevealPreimage(hash: Hash) locks value {
  clause reveal(string: String) {
    verify sha3(string) == hash
    unlock value
  }
}
`

const priceChanger = `
contract PriceChanger(askAmount: Amount, askAsset: Asset, sellerKey: PublicKey, sellerProg: Program) locks offered {
  clause changePrice(newAmount: Amount, newAsset: Asset, sig: Signature) {
    verify checkTxSig(sellerKey, sig)
    lock offered with PriceChanger(newAmount, newAsset, sellerKey, sellerProg)
  }
  clause redeem() requires payment: askAmount of askAsset {
    lock payment with sellerProg
    unlock offered
  }
}
`

const callOptionWithSettlement = `
contract CallOptionWithSettlement(strikePrice: Amount,
                    strikeCurrency: Asset,
                    sellerProgram: Program,
                    sellerKey: PublicKey,
                    buyerKey: PublicKey,
                    deadline: Time) locks underlying {
  clause exercise(buyerSig: Signature) 
                 requires payment: strikePrice of strikeCurrency {
    verify before(deadline)
    verify checkTxSig(buyerKey, buyerSig)
    lock payment with sellerProgram
    unlock underlying
  }
  clause expire() {
    verify after(deadline)
    lock underlying with sellerProgram
  }
  clause settle(sellerSig: Signature, buyerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    verify checkTxSig(buyerKey, buyerSig)
    unlock underlying
  }
}
`

func TestCompile(t *testing.T) {
	cases := []struct {
		name     string
		contract string
		want     CompileResult
	}{
		{
			"TrivialLock",
			trivialLock,
			CompileResult{
				Name:  "TrivialLock",
				Body:  mustDecodeHex("51"),
				Value: "locked",
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
			lockWithPublicKey,
			CompileResult{
				Name:  "LockWithPublicKey",
				Body:  mustDecodeHex("ae7cac"),
				Value: "locked",
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
			lockWithPKHash,
			CompileResult{
				Name:  "LockWithPublicKeyHash",
				Body:  mustDecodeHex("5279aa887cae7cac"),
				Value: "value",
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
					HashCalls: []hashCall{{
						HashType: "sha3",
						Arg:      "pubKey",
						ArgType:  "PublicKey",
					}},
				}},
			},
		},
		{
			"LockWith2of3Keys",
			lockWith2of3Keys,
			CompileResult{
				Name:  "LockWith3Keys",
				Body:  mustDecodeHex("537a547a526bae71557a536c7cad"),
				Value: "locked",
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
			lockToOutput,
			CompileResult{
				Name:  "LockToOutput",
				Body:  mustDecodeHex("0000c3c251557ac1"),
				Value: "locked",
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
			tradeOffer,
			CompileResult{
				Name:  "TradeOffer",
				Body:  mustDecodeHex("547a641300000000007251557ac16323000000547a547aae7cac690000c3c251577ac1"),
				Value: "offered",
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
			escrowedTransfer,
			CompileResult{
				Name:  "EscrowedTransfer",
				Body:  mustDecodeHex("537a641b000000537a7cae7cac690000c3c251567ac1632a000000537a7cae7cac690000c3c251557ac1"),
				Value: "value",
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
			collateralizedLoan,
			CompileResult{
				Name:  "CollateralizedLoan",
				Body:  mustDecodeHex("557a641c00000000007251567ac1695100c3c251567ac163280000007bc59f690000c3c251577ac1"),
				Value: "collateral",
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
			revealPreimage,
			CompileResult{
				Name:  "RevealPreimage",
				Body:  mustDecodeHex("7caa87"),
				Value: "value",
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
					HashCalls: []hashCall{{
						HashType: "sha3",
						Arg:      "string",
						ArgType:  "String",
					}},
				}},
			},
		},
		{
			"CallOptionWithSettlement",
			callOptionWithSettlement,
			CompileResult{
				Name:  "CallOptionWithSettlement",
				Body:  mustDecodeHex("567a76529c64390000006427000000557ac6a06971ae7cac6900007b537a51557ac16349000000557ac59f690000c3c251577ac1634900000075577a547aae7cac69557a547aae7cac"),
				Value: "underlying",
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
		{
			"PriceChanger",
			priceChanger,
			CompileResult{
				Name:  "PriceChanger",
				Body:  mustDecodeHex("557a6435000000557a5379ae7cac690000c3c251005a7989597a89597a895b7a895c7a89558902767989008901c089c1633e00000000007b537a51567ac1"),
				Value: "offered",
				Params: []ContractParam{{
					Name: "askAmount",
					Typ:  "Amount",
				}, {
					Name: "askAsset",
					Typ:  "Asset",
				}, {
					Name: "sellerKey",
					Typ:  "PublicKey",
				}, {
					Name: "sellerProg",
					Typ:  "Program",
				}},
				Clauses: []ClauseInfo{{
					Name: "changePrice",
					Args: []ClauseArg{{
						Name: "newAmount",
						Typ:  "Amount",
					}, {
						Name: "newAsset",
						Typ:  "Asset",
					}, {
						Name: "sig",
						Typ:  "Signature",
					}},
					Values: []ValueInfo{{
						Name:    "offered",
						Program: "PriceChanger(newAmount, newAsset, sellerKey, sellerProg)",
					}},
				}, {
					Name: "redeem",
					Values: []ValueInfo{{
						Name:    "payment",
						Program: "sellerProg",
						Asset:   "askAsset",
						Amount:  "askAmount",
					}, {
						Name: "offered",
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
			gotBody, _ := vm.Disassemble(got.Body, labels)
			gotProg, _ := vm.Disassemble(got.Program, nil)
			if !testutil.DeepEqual(got, c.want) {
				wantBody, _ := vm.Disassemble(c.want.Body, labels)
				wantProg, _ := vm.Disassemble(c.want.Program, nil)
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(c.want)
				t.Errorf("got  %s\nwant %s\ngot body  %s\nwant body %s\ngot prog  %s\nwant prog %s", string(gotJSON), wantJSON, gotBody, wantBody, gotProg, wantProg)
			} else {
				t.Log(gotBody)
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
