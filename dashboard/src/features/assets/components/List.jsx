import React from 'react'
import { BaseList, TableList, EmptyContent } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'asset'

const emptyListContent = <EmptyContent>
  <p>
    Assets are different types of value that may be issued and exchanged on the blockchain.
  </p>
  Learn more about how to use <a href="/docs/core/build-applications/assets" target="_blank">assets</a>.
</EmptyContent>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Asset Alias', 'Asset ID']
    },
    emptyContent: emptyListContent
  }),
  BaseList.mapDispatchToProps(type)
)
