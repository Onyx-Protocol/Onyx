import React from 'react'
import { BaseList, TableList, EmptyContent } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'mockhsm'

const firstTimeContent = <EmptyContent>
  <p>
    MockHSM keys are used for creating accounts and assets while your application is still in development.
  </p>
  <a href="/docs/core/build-applications/keys" target="_blank">Learn more</a> about how to use MockHSM keys.
</EmptyContent>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    skipQuery: true,
    label: 'MockHSM keys',
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Alias', 'xpub']
    },
    firstTimeContent
  }),
  BaseList.mapDispatchToProps(type)
)
