import accessControl from 'features/accessControl/actions'
import { actions as account } from 'features/accounts'
import { actions as app } from 'features/app'
import { actions as asset } from 'features/assets'
import { actions as balance } from 'features/balances'
import { actions as configuration } from 'features/configuration'
import { actions as core } from 'features/core'
import { actions as mockhsm } from 'features/mockhsm'
import { actions as transaction } from 'features/transactions'
import { actions as transactionFeed } from 'features/transactionFeeds'
import { actions as tutorial } from 'features/tutorial'
import { actions as unspent } from 'features/unspents'

const actions = {
  accessControl,
  account,
  app,
  asset,
  balance,
  configuration,
  core,
  mockhsm,
  transaction,
  transactionFeed,
  tutorial,
  unspent,
}

export default actions
