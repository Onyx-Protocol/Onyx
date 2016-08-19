import { combineReducers } from 'redux'
import { routerReducer } from 'react-router-redux'

import TransactionReducers from './transaction'
import UnspentReducers from './unspent'
import BalanceReducers from './balance'
import AccountReducers from './account'
import AssetReducers from './asset'
import IndexReducers from './indexQuery'
import MockHsmReducers from './mockhsm'

export default combineReducers({
  transaction: TransactionReducers,
  unspent: UnspentReducers,
  balance: BalanceReducers,
  asset: AssetReducers,
  account: AccountReducers,
  index: IndexReducers,
  routing: routerReducer,
  mockhsm: MockHsmReducers
})
