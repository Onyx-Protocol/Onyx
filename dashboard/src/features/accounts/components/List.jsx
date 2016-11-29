import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'account'

const emptyListContent = <div className="emptyContainer">
  <div className="emptyContent">
    <p>
      Accounts are used to store, receive, and transfer assets on the blockchain.
    </p>
    Learn more about how to use <a href="/docs/core/build-applications/accounts" target="_blank">accounts</a>.
  </div>
</div>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Account Alias', 'Account ID']
    },
    emptyContent: emptyListContent
  }),
  BaseList.mapDispatchToProps(type)
)
