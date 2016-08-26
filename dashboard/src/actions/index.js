import TransactionActions from './transaction'
import UnspentActions from './unspent'
import BalanceActions from './balance'
import AccountActions from './account'
import AssetActions from './asset'
import IndexActions from './indexQuery'
import MockHsmActions from './mockhsm'

export default {
  transaction: TransactionActions,
  unspent: UnspentActions,
  balance: BalanceActions,
  account: AccountActions,
  asset: AssetActions,
  index: IndexActions,
  mockhsm: MockHsmActions
}
