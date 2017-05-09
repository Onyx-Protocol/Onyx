import { mustCompileTemplate } from './util'
import { client } from '../util'
import { Item, ItemMap, State } from './types'
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

export const itemMap: ItemMap = {}
export const INITIAL_STATE: State = { itemMap: {}, idList: [], source: '', selected: '' }
