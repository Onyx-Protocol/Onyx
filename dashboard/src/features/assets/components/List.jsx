import React from 'react'
import { BaseList, TableList, EmptyContent } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'asset'

const emptyListContent = <EmptyContent>
  <p>
    Assets are different types of value that may be issued and exchanged on the blockchain.
  </p>
  <a href="/docs/core/build-applications/assets" target="_blank">Learn more</a> about how to use assets.
</EmptyContent>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Asset Alias', 'Asset ID']
    },
    firstTimeContent: emptyListContent
  }),
  BaseList.mapDispatchToProps(type)
)
