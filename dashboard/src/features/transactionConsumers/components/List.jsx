import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'
import { actions } from 'features/transactionConsumers'

const type = 'transactionConsumer'

const dispatch = (dispatch) => ({
  ...BaseList.mapDispatchToProps(type)(dispatch),
  updateQuery: null,
  itemActions: {
    delete: (id) => dispatch(actions.deleteItem(id))
  },
})

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    skipQuery: true,
    label: 'Transaction Consumers'
  }),
  dispatch
)
