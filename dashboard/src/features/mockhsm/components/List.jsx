import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'mockhsm'

const emptyListContent = <div className="emptyContainer">
  <p>
    MockHSM keys are used for creating accounts and assets while your application is still in development.
  </p>
  Learn more about how to use <a href="/docs/core/build-applications/keys" target="_blank">MockHSM keys</a>.
</div>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    skipQuery: true,
    label: 'MockHSM keys',
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Alias', 'xpub']
    },
    emptyContent: emptyListContent
  }),
  BaseList.mapDispatchToProps(type)
)
