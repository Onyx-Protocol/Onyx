package ivy

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const trivialLock = `
contract TrivialLock(locked: Value) {
  clause unlock() {
    return locked
  }
}
`

const lockWithPublicKey = `
contract LockWithPublicKey(publicKey: PublicKey, locked: Value) {
  clause unlock(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    return locked
  }
}
`

const lockWith2of3Keys = `
contract LockWith2of3Keys(pubkey1, pubkey2, pubkey3: PublicKey, locked: Value) {
  clause unlock(sig1, sig2: Signature) {
    verify checkTxMultiSig([pubkey1, pubkey2, pubkey3], [sig1, sig2])
    return locked
  }
}
`

const lockToOutput = `
contract LockToOutput(address: Address, locked: Value) {
  clause unlock() {
    output address(locked)
  }
}
`

const tradeOffer = `
contract TradeOffer(requested: AssetAmount, sellerAddress: Address, sellerKey: PublicKey, offered: Value) {
  clause trade(payment: Value) {
    verify payment.assetAmount == requested
    output sellerAddress(payment)
    return offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    output sellerAddress(offered)
  }
}
`

const escrowedTransfer = `
contract EscrowedTransfer(
  agent: PublicKey,
  sender: Address,
  recipient: Address,
  value: Value
) {
  clause approve(sig: Signature) {
    verify checkTxSig(agent, sig)
    output recipient(value)
  }
  clause reject(sig: Signature) {
    verify checkTxSig(agent, sig)
    output sender(value)
  }
}
`

const collateralizedLoan = `
contract CollateralizedLoan(
  balance: AssetAmount,
  deadline: Time,
  lender: Address,
  borrower: Address,
  collateral: Value
) {
  clause repay(payment: Value) {
    verify payment.assetAmount == balance
    output lender(payment)
    output borrower(collateral)
  }
  clause default() {
    verify after(deadline)
    output lender(collateral)
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
				Params: []ContractParam{{
					Name: "locked",
					Typ:  "Value",
				}},
				Clauses: []ClauseInfo{{
					Name: "unlock",
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
				Program: mustDecodeHex("6eae7cac"),
				Params: []ContractParam{{
					Name: "publicKey",
					Typ:  "PublicKey",
				}, {
					Name: "locked",
					Typ:  "Value",
				}},
				Clauses: []ClauseInfo{{
					Name: "unlock",
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
			"LockWith2of3Keys",
			lockWith2of3Keys,
			CompileResult{
				Name:    "LockWith2of3Keys",
				Program: mustDecodeHex("54795479526bae557955795579536c7cad"),
				Params: []ContractParam{{
					Name: "pubkey1",
					Typ:  "PublicKey",
				}, {
					Name: "pubkey2",
					Typ:  "PublicKey",
				}, {
					Name: "pubkey3",
					Typ:  "PublicKey",
				}, {
					Name: "locked",
					Typ:  "Value",
				}},
				Clauses: []ClauseInfo{{
					Name: "unlock",
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
				Program: mustDecodeHex("0000c3c2515579c1"),
				Params: []ContractParam{{
					Name: "address",
					Typ:  "Address",
				}, {
					Name: "locked",
					Typ:  "Value",
				}},
				Clauses: []ClauseInfo{{
					Name: "unlock",
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
				Program: mustDecodeHex("547a6413000000000070515779c1632300000054795479ae7cac690000c3c2515779c1"),
				Params: []ContractParam{{
					Name: "requested",
					Typ:  "AssetAmount",
				}, {
					Name: "sellerAddress",
					Typ:  "Address",
				}, {
					Name: "sellerKey",
					Typ:  "PublicKey",
				}, {
					Name: "offered",
					Typ:  "Value",
				}},
				Clauses: []ClauseInfo{{
					Name: "trade",
					Args: []ClauseArg{{
						Name: "payment",
						Typ:  "Value",
					}},
					Values: []ValueInfo{{
						Name:        "payment",
						Program:     "sellerAddress",
						AssetAmount: "requested",
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
						Program: "sellerAddress",
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
				Program: mustDecodeHex("537a641b000000537978ae7cac690000c3c2515779c1632a000000537978ae7cac690000c3c2515679c1"),
				Params: []ContractParam{{
					Name: "agent",
					Typ:  "PublicKey",
				}, {
					Name: "sender",
					Typ:  "Address",
				}, {
					Name: "recipient",
					Typ:  "Address",
				}, {
					Name: "value",
					Typ:  "Value",
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
				Program: mustDecodeHex("557a641c000000000070515879c1695100c3c2515979c163290000005279c59f690000c3c2515879c1"),
				Params: []ContractParam{{
					Name: "balance",
					Typ:  "AssetAmount",
				}, {
					Name: "deadline",
					Typ:  "Time",
				}, {
					Name: "lender",
					Typ:  "Address",
				}, {
					Name: "borrower",
					Typ:  "Address",
				}, {
					Name: "collateral",
					Typ:  "Value",
				}},
				Clauses: []ClauseInfo{{
					Name: "repay",
					Args: []ClauseArg{{
						Name: "payment",
						Typ:  "Value",
					}},
					Values: []ValueInfo{
						{
							Name:        "payment",
							Program:     "lender",
							AssetAmount: "balance",
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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.contract)
			got, err := Compile(r, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(c.want)
				t.Errorf("got %s\nwant %s", string(gotJSON), wantJSON)
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
