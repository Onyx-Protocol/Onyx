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

export const LOCK_TO_OUTPUT =`contract LockToOutput(address: Address, locked: Value) {
  clause unlock() {
    output address(locked)
  }
}`

export const TRADE_OFFER = `contract TradeOffer(
  requested: AssetAmount,
  sellerAddress: Address,
  sellerKey: PublicKey,
  offered: Value
) {
  clause trade(payment: Value) {
    verify payment.assetAmount == requested
    output sellerAddress(payment)
    return offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    output sellerAddress(offered)
  }
}`

export const ESCROWED_TRANSFER = `contract EscrowedTransfer(
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
}`

export const COLLATERALIZED_LOAN = `contract CollateralizedLoan(
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
  LockToOutput: LOCK_TO_OUTPUT,
  TradeOffer: TRADE_OFFER,
  EscrowedTransfer: ESCROWED_TRANSFER,
  CollateralizedLoan: COLLATERALIZED_LOAN,
  RevealPreimage: REVEAL_PREIMAGE,
  RevealFactors: REVEAL_FACTORS
}

export const idList = [
  "TrivialLock",
  "LockWithPublicKey",
  "LockToOutput",
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
