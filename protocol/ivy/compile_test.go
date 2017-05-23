package ivy

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
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
		wantJSON string
	}{
		{
			"TrivialLock",
			trivialLock,
			`[{"name":"TrivialLock","clauses":[{"name":"trivialUnlock","values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"51","body_opcodes":"TRUE","program":"00015100c0"}]`,
		},
		{
			"LockWithPublicKey",
			lockWithPublicKey,
			`[{"name":"LockWithPublicKey","params":[{"name":"publicKey","declared_type":"PublicKey"}],"clauses":[{"name":"unlockWithSig","params":[{"name":"sig","declared_type":"Signature"}],"values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"ae7cac","body_opcodes":"TXSIGHASH SWAP CHECKSIG"}]`,
		},
		{
			"LockWithPublicKeyHash",
			lockWithPKHash,
			`[{"name":"LockWithPublicKeyHash","params":[{"name":"pubKeyHash","declared_type":"Hash","inferred_type":"Sha3(PublicKey)"}],"clauses":[{"name":"spend","params":[{"name":"pubKey","declared_type":"PublicKey"},{"name":"sig","declared_type":"Signature"}],"hash_calls":[{"hash_type":"sha3","arg":"pubKey","arg_type":"PublicKey"}],"values":[{"name":"value"}]}],"value":"value","body_bytecode":"5279aa887cae7cac","body_opcodes":"2 PICK SHA3 EQUALVERIFY SWAP TXSIGHASH SWAP CHECKSIG"}]`,
		},
		{
			"LockWith2of3Keys",
			lockWith2of3Keys,
			`[{"name":"LockWith3Keys","params":[{"name":"pubkey1","declared_type":"PublicKey"},{"name":"pubkey2","declared_type":"PublicKey"},{"name":"pubkey3","declared_type":"PublicKey"}],"clauses":[{"name":"unlockWith2Sigs","params":[{"name":"sig1","declared_type":"Signature"},{"name":"sig2","declared_type":"Signature"}],"values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"537a547a526bae71557a536c7cad","body_opcodes":"3 ROLL 4 ROLL 2 TOALTSTACK TXSIGHASH 2ROT 5 ROLL 3 FROMALTSTACK SWAP CHECKMULTISIG"}]`,
		},
		{
			"LockToOutput",
			lockToOutput,
			`[{"name":"LockToOutput","params":[{"name":"address","declared_type":"Program"}],"clauses":[{"name":"relock","values":[{"name":"locked","program":"address"}]}],"value":"locked","body_bytecode":"0000c3c251557ac1","body_opcodes":"0 0 AMOUNT ASSET 1 5 ROLL CHECKOUTPUT"}]`,
		},
		{
			"TradeOffer",
			tradeOffer,
			`[{"name":"TradeOffer","params":[{"name":"requestedAsset","declared_type":"Asset"},{"name":"requestedAmount","declared_type":"Amount"},{"name":"sellerProgram","declared_type":"Program"},{"name":"sellerKey","declared_type":"PublicKey"}],"clauses":[{"name":"trade","reqs":[{"name":"payment","asset":"requestedAsset","amount":"requestedAmount"}],"values":[{"name":"payment","program":"sellerProgram","asset":"requestedAsset","amount":"requestedAmount"},{"name":"offered"}]},{"name":"cancel","params":[{"name":"sellerSig","declared_type":"Signature"}],"values":[{"name":"offered","program":"sellerProgram"}]}],"value":"offered","body_bytecode":"547a641300000000007251557ac16323000000547a547aae7cac690000c3c251577ac1","body_opcodes":"4 ROLL JUMPIF:$cancel $trade 0 0 2SWAP 1 5 ROLL CHECKOUTPUT JUMP:$_end $cancel 4 ROLL 4 ROLL TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 7 ROLL CHECKOUTPUT $_end"}]`,
		},
		{
			"EscrowedTransfer",
			escrowedTransfer,
			`[{"name":"EscrowedTransfer","params":[{"name":"agent","declared_type":"PublicKey"},{"name":"sender","declared_type":"Program"},{"name":"recipient","declared_type":"Program"}],"clauses":[{"name":"approve","params":[{"name":"sig","declared_type":"Signature"}],"values":[{"name":"value","program":"recipient"}]},{"name":"reject","params":[{"name":"sig","declared_type":"Signature"}],"values":[{"name":"value","program":"sender"}]}],"value":"value","body_bytecode":"537a641b000000537a7cae7cac690000c3c251567ac1632a000000537a7cae7cac690000c3c251557ac1","body_opcodes":"3 ROLL JUMPIF:$reject $approve 3 ROLL SWAP TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 6 ROLL CHECKOUTPUT JUMP:$_end $reject 3 ROLL SWAP TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 5 ROLL CHECKOUTPUT $_end"}]`,
		},
		{
			"CollateralizedLoan",
			collateralizedLoan,
			`[{"name":"CollateralizedLoan","params":[{"name":"balanceAsset","declared_type":"Asset"},{"name":"balanceAmount","declared_type":"Amount"},{"name":"deadline","declared_type":"Time"},{"name":"lender","declared_type":"Program"},{"name":"borrower","declared_type":"Program"}],"clauses":[{"name":"repay","reqs":[{"name":"payment","asset":"balanceAsset","amount":"balanceAmount"}],"values":[{"name":"payment","program":"lender","asset":"balanceAsset","amount":"balanceAmount"},{"name":"collateral","program":"borrower"}]},{"name":"default","mintimes":["deadline"],"values":[{"name":"collateral","program":"lender"}]}],"value":"collateral","body_bytecode":"557a641c00000000007251567ac1695100c3c251567ac163280000007bc59f690000c3c251577ac1","body_opcodes":"5 ROLL JUMPIF:$default $repay 0 0 2SWAP 1 6 ROLL CHECKOUTPUT VERIFY 1 0 AMOUNT ASSET 1 6 ROLL CHECKOUTPUT JUMP:$_end $default ROT MINTIME LESSTHAN VERIFY 0 0 AMOUNT ASSET 1 7 ROLL CHECKOUTPUT $_end"}]`,
		},
		{
			"RevealPreimage",
			revealPreimage,
			`[{"name":"RevealPreimage","params":[{"name":"hash","declared_type":"Hash","inferred_type":"Sha3(String)"}],"clauses":[{"name":"reveal","params":[{"name":"string","declared_type":"String"}],"hash_calls":[{"hash_type":"sha3","arg":"string","arg_type":"String"}],"values":[{"name":"value"}]}],"value":"value","body_bytecode":"7caa87","body_opcodes":"SWAP SHA3 EQUAL"}]`,
		},
		{
			"CallOptionWithSettlement",
			callOptionWithSettlement,
			`[{"name":"CallOptionWithSettlement","params":[{"name":"strikePrice","declared_type":"Amount"},{"name":"strikeCurrency","declared_type":"Asset"},{"name":"sellerProgram","declared_type":"Program"},{"name":"sellerKey","declared_type":"PublicKey"},{"name":"buyerKey","declared_type":"PublicKey"},{"name":"deadline","declared_type":"Time"}],"clauses":[{"name":"exercise","params":[{"name":"buyerSig","declared_type":"Signature"}],"reqs":[{"name":"payment","asset":"strikeCurrency","amount":"strikePrice"}],"maxtimes":["deadline"],"values":[{"name":"payment","program":"sellerProgram","asset":"strikeCurrency","amount":"strikePrice"},{"name":"underlying"}]},{"name":"expire","mintimes":["deadline"],"values":[{"name":"underlying","program":"sellerProgram"}]},{"name":"settle","params":[{"name":"sellerSig","declared_type":"Signature"},{"name":"buyerSig","declared_type":"Signature"}],"values":[{"name":"underlying"}]}],"value":"underlying","body_bytecode":"567a76529c64390000006427000000557ac6a06971ae7cac6900007b537a51557ac16349000000557ac59f690000c3c251577ac1634900000075577a547aae7cac69557a547aae7cac","body_opcodes":"6 ROLL DUP 2 NUMEQUAL JUMPIF:$settle JUMPIF:$expire $exercise 5 ROLL MAXTIME GREATERTHAN VERIFY 2ROT TXSIGHASH SWAP CHECKSIG VERIFY 0 0 ROT 3 ROLL 1 5 ROLL CHECKOUTPUT JUMP:$_end $expire 5 ROLL MINTIME LESSTHAN VERIFY 0 0 AMOUNT ASSET 1 7 ROLL CHECKOUTPUT JUMP:$_end $settle DROP 7 ROLL 4 ROLL TXSIGHASH SWAP CHECKSIG VERIFY 5 ROLL 4 ROLL TXSIGHASH SWAP CHECKSIG $_end"}]`,
		},
		{
			"PriceChanger",
			priceChanger,
			`[{"name":"PriceChanger","params":[{"name":"askAmount","declared_type":"Amount"},{"name":"askAsset","declared_type":"Asset"},{"name":"sellerKey","declared_type":"PublicKey"},{"name":"sellerProg","declared_type":"Program"}],"clauses":[{"name":"changePrice","params":[{"name":"newAmount","declared_type":"Amount"},{"name":"newAsset","declared_type":"Asset"},{"name":"sig","declared_type":"Signature"}],"values":[{"name":"offered","program":"PriceChanger(newAmount, newAsset, sellerKey, sellerProg)"}]},{"name":"redeem","reqs":[{"name":"payment","asset":"askAsset","amount":"askAmount"}],"values":[{"name":"payment","program":"sellerProg","asset":"askAsset","amount":"askAmount"},{"name":"offered"}]}],"value":"offered","body_bytecode":"557a6435000000557a5379ae7cac690000c3c251005a7a89597a89587a895a7a895b7a89558902767989008901c089c1633e00000000007b537a51567ac1","body_opcodes":"5 ROLL JUMPIF:$redeem $changePrice 5 ROLL 3 PICK TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 0 10 ROLL CATPUSHDATA 9 ROLL CATPUSHDATA 8 ROLL CATPUSHDATA 10 ROLL CATPUSHDATA 11 ROLL CATPUSHDATA 5 CATPUSHDATA 0x7679 CATPUSHDATA 0 CATPUSHDATA 192 CATPUSHDATA CHECKOUTPUT JUMP:$_end $redeem 0 0 ROT 3 ROLL 1 6 ROLL CHECKOUTPUT $_end"}]`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.contract)
			got, err := Compile(r, nil)
			if err != nil {
				t.Fatal(err)
			}
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != c.wantJSON {
				t.Errorf("\ngot  %s\nwant %s", string(gotJSON), c.wantJSON)
			} else {
				for _, contract := range got {
					t.Log(contract.Opcodes)
				}
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
