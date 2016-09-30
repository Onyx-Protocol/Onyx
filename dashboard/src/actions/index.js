import { actions as accessToken } from '../features/accessTokens'
import { actions as account } from '../features/accounts'
import { actions as app } from '../features/app'
import { actions as asset } from '../features/assets'
import { actions as transaction } from '../features/transactions'
import balance from './balance'
import core from './core'
import mockhsm from './mockhsm'
import routing from './routing'
import unspent from './unspent'

const actions = {
  ...accessToken,
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

export default actions
