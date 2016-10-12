import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'
import { actions } from 'features/transactionFeeds'

const type = 'transactionFeed'

const dispatch = (dispatch) => ({
  ...BaseList.mapDispatchToProps(type)(dispatch),
  itemActions: {
    delete: (id) => dispatch(actions.deleteItem(id))
  },
})

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    skipQuery: true,
    label: 'Transaction Feeds'
  }),
  dispatch
)
