import { baseListActions, baseFormActions } from 'features/shared/actions'

const type = 'transactionFeed'

export default {
  ...baseFormActions(type, {
    listPath: 'transaction-feeds',
    className: 'TransactionFeed',
  }),
  ...baseListActions(type, {
    listPath: 'transaction-feeds',
    className: 'TransactionFeed',
  }),
}
