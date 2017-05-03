import { mustCompileTemplate } from './util'
import { Item, State } from './types'
import { compileTemplate } from 'ivy-compiler'

export const NAME = 'templates'

const TRIVIAL_LOCK =`contract TrivialLock(locked: Value) {
  clause unlock() {
    return locked
  }
}`

const LOCK_WITH_PUBLIC_KEY = `contract LockWithPublicKey(publicKey: PublicKey, locked: Value) {
  clause unlock(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    return locked
  }
}`

const LOCK_TO_OUTPUT =`contract LockToOutput(program: Program, locked: Value) {
  clause unlock() {
    output program(locked)
  }
}`

const TRADE_OFFER = `contract TradeOffer(
  requested: AssetAmount, 
  sellerControlProgram: Program, 
  sellerKey: PublicKey, 
  offered: Value
) {
  clause trade(payment: Value) {
    verify payment.assetAmount == requested
    output sellerControlProgram(payment)
    return offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    output sellerControlProgram(offered)
  }
}`

const ESCROWED_TRANSFER = `contract EscrowedTransfer(
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

export const itemMap = {}
itemMap["TrivialLock"] = mustCompileTemplate(TRIVIAL_LOCK) as Item
itemMap["LockWithPublicKey"] = mustCompileTemplate(LOCK_WITH_PUBLIC_KEY) as Item
itemMap["LockToOutput"] = mustCompileTemplate(LOCK_TO_OUTPUT) as Item
itemMap["TradeOffer"] = mustCompileTemplate(TRADE_OFFER) as Item
itemMap["EscrowedTransfer"] = mustCompileTemplate(ESCROWED_TRANSFER) as Item
const idList = ["TrivialLock", "LockWithPublicKey", "LockToOutput", "TradeOffer", "EscrowedTransfer"]
const source = itemMap["TrivialLock"].source
const selected = idList[0]
export const INITIAL_STATE: State = { itemMap, idList, source, selected }