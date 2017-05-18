// Predefined contract templates

export const BASE_TEMPLATE = `contract ContractName() locks value {
  clause clauseName() {
    unlock value
  }
}`

export const LOCK_WITH_PUBLIC_KEY = `contract LockWithPublicKey(publicKey: PublicKey) locks value {
  clause spend(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    unlock value
  }
}`

export const LOCK_WITH_PUBLIC_KEY_HASH = `contract LockWithPublicKeyHash(pubKeyHash: Hash) locks value {
  clause spend(pubKey: PublicKey, sig: Signature) {
    verify sha3(pubKey) == pubKeyHash
    verify checkTxSig(pubKey, sig)
    unlock value
  }
}`

export const LOCK_WITH_MULTISIG = `contract LockWithMultiSig(publicKey1: PublicKey,
                          publicKey2: PublicKey,
                          publicKey3: PublicKey) locks value {
  clause spend(sig1: Signature, sig2: Signature) {
    verify checkTxMultiSig([publicKey1, publicKey2, publicKey3], [sig1, sig2])
    unlock value
  }
}`

export const TRADE_OFFER = `contract TradeOffer(assetRequested: Asset,
                    amountRequested: Amount,
                    seller: Program,
                    cancelKey: PublicKey) locks offered {
  clause trade() requires payment: amountRequested of assetRequested {
    lock payment with seller
    unlock offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(cancelKey, sellerSig)
    unlock offered
  }
}`

export const ESCROW = `contract Escrow(agent: PublicKey,
                sender: Program,
                recipient: Program) locks value {
  clause approve(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock value with recipient
  }
  clause reject(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock value with sender
  }
}`

export const LOAN_COLLATERAL =`contract LoanCollateral(assetLoaned: Asset,
                        amountLoaned: Amount,
                        repaymentDue: Time,
                        lender: Program,
                        borrower: Program) locks collateral {
  clause repay() requires payment: amountLoaned of assetLoaned {
    lock payment with lender
    lock collateral with borrower
  }
  clause default() {
    verify after(repaymentDue)
    lock collateral with lender
  }
}`

export const REVEAL_PREIMAGE = `contract RevealPreimage(hash: Hash) locks value {
  clause reveal(string: String) {
    verify sha3(string) == hash
    unlock value
  }
}`

export const REVEAL_FACTORS = `contract RevealFactors(product: Integer) locks value {
  clause reveal(factor1: Integer, factor2: Integer) {
    verify factor1 * factor2 == product
    unlock value
  }
}`

export const CALL_OPTION = `contract CallOption(strikePrice: Amount,
                    strikeCurrency: Asset,
                    seller: Program,
                    buyerKey: PublicKey,
                    deadline: Time) locks underlying {
  clause exercise(buyerSig: Signature) requires payment: strikePrice of strikeCurrency {
    verify before(deadline)
    verify checkTxSig(buyerKey, buyerSig)
    lock payment with seller
    unlock underlying
  }
  clause expire() {
    verify after(deadline)
    lock underlying with seller
  }
}`

export const INITIAL_SOURCE_MAP = {
  ContractName: BASE_TEMPLATE,
  LockWithPublicKey: LOCK_WITH_PUBLIC_KEY,
  LockWithPublicKeyHash: LOCK_WITH_PUBLIC_KEY_HASH,
  LockWithMultiSig: LOCK_WITH_MULTISIG,
  TradeOffer: TRADE_OFFER,
  Escrow: ESCROW,
  LoanCollateral: LOAN_COLLATERAL,
  RevealPreimage: REVEAL_PREIMAGE,
  RevealFactors: REVEAL_FACTORS,
  CallOption: CALL_OPTION
}

export const INITIAL_ID_LIST = [
  "ContractName",
  "LockWithPublicKey",
  "LockWithPublicKeyHash",
  "LockWithMultiSig",
  "TradeOffer",
  "Escrow",
  "LoanCollateral",
  "CallOption",
  "RevealPreimage",
]
