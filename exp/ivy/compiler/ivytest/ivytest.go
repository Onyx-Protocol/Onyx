package ivytest

const TrivialLock = `
contract TrivialLock() locks locked {
  clause trivialUnlock() {
    unlock locked
  }
}
`

const LockWithPublicKey = `
contract LockWithPublicKey(publicKey: PublicKey) locks locked {
  clause unlockWithSig(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    unlock locked
  }
}
`

const LockWithPKHash = `
contract LockWithPublicKeyHash(pubKeyHash: Hash) locks value {
  clause spend(pubKey: PublicKey, sig: Signature) {
    verify sha3(pubKey) == pubKeyHash
    verify checkTxSig(pubKey, sig)
    unlock value
  }
}
`

const LockWith2of3Keys = `
contract LockWith3Keys(pubkey1, pubkey2, pubkey3: PublicKey) locks locked {
  clause unlockWith2Sigs(sig1, sig2: Signature) {
    verify checkTxMultiSig([pubkey1, pubkey2, pubkey3], [sig1, sig2])
    unlock locked
  }
}
`

const LockToOutput = `
contract LockToOutput(address: Program) locks locked {
  clause relock() {
    lock locked with address
  }
}
`

const TradeOffer = `
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

const EscrowedTransfer = `
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

const CollateralizedLoan = `
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

const RevealPreimage = `
contract RevealPreimage(hash: Hash) locks value {
  clause reveal(string: String) {
    verify sha3(string) == hash
    unlock value
  }
}
`

const PriceChanger = `
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

const CallOptionWithSettlement = `
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

const OneTwo = `
contract Two(b, c: Program, expirationTime: Time) locks value {
  clause redeem() {
    verify before(expirationTime)
    lock value with b
  }
  clause default() {
    verify after(expirationTime)
    lock value with c
  }
}
contract One(a, b, c: Program, switchTime, expirationTime: Time) locks value {
  clause redeem() {
    verify before(switchTime)
    lock value with a
  }
  clause switch() {
    verify after(switchTime)
    lock value with Two(b, c, expirationTime)
  }
}
`
