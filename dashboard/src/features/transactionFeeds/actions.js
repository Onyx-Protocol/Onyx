import { baseListActions, baseFormActions } from 'features/shared/actions'

const type = 'transactionFeed'

export default {
  ...baseFormActions(type, {
    listPath: 'transactions/feeds',
    className: 'TransactionFeed',
  }),
  ...baseListActions(type, {
    listPath: 'transactions/feeds',
    className: 'TransactionFeed',
  }),
}
