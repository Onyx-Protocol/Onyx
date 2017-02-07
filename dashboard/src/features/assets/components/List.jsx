import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'asset'

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Asset Alias', 'Asset ID']
    }
  }),
  BaseList.mapDispatchToProps(type)
)
