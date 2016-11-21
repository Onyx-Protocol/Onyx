import React from 'react'
import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'
import { actions } from 'features/transactionFeeds'

const type = 'transactionFeed'

let EmptyContent = <div>
  <p>
    Transaction feeds can be used to process transactions as they arrive on the
    blockchain. This is helpful for real-time applications such as notifications
    or live-updating interfaces.
  </p>
  <p>
    Learn more about how transaction feeds can eliminate the need for polling or
    keeping state in your application by checking out the&nbsp;
    <a href="/docs/core/build-applications/real-time-transaction-processing" target="_blank">Real-time Transaction Processing</a> guide
    in the documentation.
  </p>
</div>

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
    label: 'transaction feeds',
    emptyContent: EmptyContent
  }),
  dispatch
)
