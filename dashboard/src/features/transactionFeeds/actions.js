import { baseListActions, baseCreateActions } from 'features/shared/actions'

const type = 'transactionFeed'

export default {
  ...baseCreateActions(type, {
    listPath: 'transaction-feeds',
    className: 'TransactionFeed',
  }),
  ...baseListActions(type, {
    listPath: 'transaction-feeds',
    className: 'TransactionFeed',
  }),
}
