import { actions as account } from '../features/accounts'
import { actions as asset } from '../features/assets'
import { actions as transaction } from '../features/transactions'
import app from './app'
import balance from './balance'
import core from './core'
import mockhsm from './mockhsm'
import routing from './routing'
import unspent from './unspent'

export default {
  account,
  app,
  asset,
  balance,
  core,
  mockhsm,
  routing,
  transaction,
  unspent,
}
