import { baseListActions, baseFormActions } from 'features/shared/actions'

const type = 'transactionConsumer'

export default {
  ...baseFormActions(type, {
    listPath: 'transactions/consumers',
    className: 'TransactionConsumer',
  }),
  ...baseListActions(type, {
    listPath: 'transactions/consumers',
    className: 'TransactionConsumer',
  }),
}
