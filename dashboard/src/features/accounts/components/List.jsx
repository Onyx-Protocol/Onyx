import React from 'react'
import { BaseList, TableList, EmptyContent } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'account'

const emptyListContent = <EmptyContent>
  <p>
    Accounts are used to store, receive, and transfer assets on the blockchain.
  </p>
  <a href="/docs/core/build-applications/accounts" target="_blank">Learn more</a> about how to use accounts.
</EmptyContent>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Account Alias', 'Account ID']
    },
    firstTimeContent: emptyListContent
  }),
  BaseList.mapDispatchToProps(type)
)
