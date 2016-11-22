import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'account'

const EmptyContent = <div className="emptyContainer">
  <p>
    An account is an object in Chain Core that tracks ownership
    of assets on a blockchain by creating and tracking control programs.
  </p>
  <p>
    Learn more about how to create accounts and control programs by checking
    out the <a href="/docs/core/build-applications/accounts" target="_blank">Accounts</a> guide
    in the documentation.
  </p>
</div>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Account Alias', 'Account ID']
    },
    emptyContent: EmptyContent
  }),
  BaseList.mapDispatchToProps(type)
)
