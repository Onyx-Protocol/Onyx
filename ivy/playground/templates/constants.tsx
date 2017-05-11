import { client } from '../util'
import { Template, ItemMap, TemplateState } from './types'
import { compileTemplate } from 'ivy-compiler'

export const NAME = 'templates'

export const TRIVIAL_LOCK =`contract TrivialLock(locked: Value) {
  clause unlock() {
    return locked
  }
}`

export const LOCK_WITH_PUBLIC_KEY = `contract LockWithPublicKey(publicKey: PublicKey, locked: Value) {
  clause unlock(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    return locked
  }
}`

export const LOCK_WITH_PUBLIC_KEY_HASH = `contract LockWithPublicKeyHash(pubKeyHash: Hash, locked: Value) {
  clause unlock(pubKey: PublicKey, sig: Signature) {
    verify sha3(pubKey) == pubKeyHash
    verify checkTxSig(pubKey, sig)
    return locked
  }
}`

export const LOCK_WITH_MULTISIG = `contract LockWithMultiSig(
  publicKey1: PublicKey, 
  publicKey2: PublicKey, 
  publicKey3: PublicKey,
  locked: Value) {
  clause unlock(sig1: Signature, sig2: Signature) {
    verify checkTxMultiSig([publicKey1, publicKey2, publicKey3], [sig1, sig2])
    return locked
  }
}`

export const TRADE_OFFER = `contract TradeOffer(
  requested: AssetAmount,
  sellerProgram: Program,
  sellerKey: PublicKey,
  offered: Value
) {
  clause trade(payment: Value) {
    verify payment.assetAmount == requested
    output sellerProgram(payment)
    return offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    output sellerProgram(offered)
  }
}`

export const ESCROWED_TRANSFER = `contract EscrowedTransfer(
  agent: PublicKey,
  sender: Program,
  recipient: Program,
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
}`

export const COLLATERALIZED_LOAN = `contract CollateralizedLoan(
  balance: AssetAmount,
  deadline: Time,
  lender: Program,
  borrower: Program,
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
}`

export const REVEAL_PREIMAGE = `contract RevealPreimage(hash: Hash, value: Value) {
  clause reveal(string: String) {
    verify sha3(string) == hash
    return value
  }
}`

export const REVEAL_FACTORS = `contract RevealFactors(product: Integer, value: Value) {
  clause reveal(factor1: Integer, factor2: Integer) {
    verify factor1 * factor2 == product
    return value
  }
}`

const itemMap = {
  TrivialLock: TRIVIAL_LOCK,
  LockWithPublicKey: LOCK_WITH_PUBLIC_KEY,
  LockWithPublicKeyHash: LOCK_WITH_PUBLIC_KEY_HASH,
  LockWithMultiSig: LOCK_WITH_MULTISIG,
  TradeOffer: TRADE_OFFER,
  EscrowedTransfer: ESCROWED_TRANSFER,
  CollateralizedLoan: COLLATERALIZED_LOAN,
  RevealPreimage: REVEAL_PREIMAGE,
  RevealFactors: REVEAL_FACTORS
}

export const idList = [
  "TrivialLock",
  "LockWithPublicKey",
  "LockWithPublicKeyHash",
  "LockWithMultiSig",
  "TradeOffer",
  "EscrowedTransfer",
  "CollateralizedLoan",
  "RevealPreimage"
]

const selected = idList[0]
const source = itemMap[selected].source

export const INITIAL_STATE: TemplateState = { 
  itemMap, 
  idList, 
  source: '',
  inputMap: {},
  compiled: undefined
}
