import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'
import { actions } from 'features/transactionFeeds'

const type = 'transactionFeed'

const dispatch = (dispatch) => ({
  ...BaseList.mapDispatchToProps(type)(dispatch),
  itemActions: {
    delete: (feed) => {
      let label = `ID ${feed.id}`
      if (!!feed.alias && feed.alias.length > 0) {
        label = `"${feed.alias}"`
      }

      dispatch(actions.deleteItem(
        feed.id,
        `Really delete transaction feed ${label}?`,
        `Deleted transaction feed ${label}.`
      ))
    }
  },
})

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    skipQuery: true,
    label: 'transaction feeds'
  }),
  dispatch
)
