package ivy

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"chain/protocol/vm"
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
				Name:    "TrivialLock",
				Program: mustDecodeHex("51"),
				Value:   "locked",
				Params:  []ContractParam{},
				Clauses: []ClauseInfo{{
					Name: "trivialUnlock",
					Args: []ClauseArg{},
					Values: []ValueInfo{{
						Name: "locked",
					}},
					Mintimes: []string{},
					Maxtimes: []string{},
				}},
			},
		},
		{
			"LockWithPublicKey",
			lockWithPublicKey,
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
					Mintimes: []string{},
					Maxtimes: []string{},
				}},
			},
		},
		{
			"LockWithPublicKeyHash",
			lockWithPKHash,
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
					Mintimes: []string{},
					Maxtimes: []string{},
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
					Mintimes: []string{},
					Maxtimes: []string{},
				}},
			},
		},
		{
			"LockToOutput",
			lockToOutput,
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
					Args: []ClauseArg{},
					Values: []ValueInfo{{
						Name:    "locked",
						Program: "address",
					}},
					Mintimes: []string{},
					Maxtimes: []string{},
				}},
			},
		},
		{
			"TradeOffer",
			tradeOffer,
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
					Args: []ClauseArg{},
					Values: []ValueInfo{{
						Name:    "payment",
						Program: "sellerProgram",
						Asset:   "requestedAsset",
						Amount:  "requestedAmount",
					}, {
						Name: "offered",
					}},
					Mintimes: []string{},
					Maxtimes: []string{},
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
					Mintimes: []string{},
					Maxtimes: []string{},
				}},
			},
		},
		{
			"EscrowedTransfer",
			escrowedTransfer,
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
					Mintimes: []string{},
					Maxtimes: []string{},
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
					Mintimes: []string{},
					Maxtimes: []string{},
				}},
			},
		},
		{
			"CollateralizedLoan",
			collateralizedLoan,
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
					Args: []ClauseArg{},
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
					Mintimes: []string{},
					Maxtimes: []string{},
				}, {
					Name: "default",
					Args: []ClauseArg{},
					Values: []ValueInfo{
						{
							Name:    "collateral",
							Program: "lender",
						},
					},
					Mintimes: []string{"deadline"},
					Maxtimes: []string{},
				}},
			},
		},
		{
			"RevealPreimage",
			revealPreimage,
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
					Mintimes: []string{},
					Maxtimes: []string{},
					HashCalls: []hashCall{{
						HashType: "sha3",
						Arg:      "string",
						ArgType:  "String",
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
			gotProg, _ := vm.Disassemble(got.Program)
			if !reflect.DeepEqual(got, c.want) {
				wantProg, _ := vm.Disassemble(c.want.Program)
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
