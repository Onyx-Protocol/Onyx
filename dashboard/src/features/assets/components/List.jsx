import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'asset'

const emptyListContent = <div className="emptyContainer">
  <div className="emptyContent">
    <p>
      Assets are different types of value that may be issued and exchanged on the blockchain.
    </p>
    Learn more about how to use <a href="/docs/core/build-applications/assets" target="_blank">assets</a>.
  </div>
</div>

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
