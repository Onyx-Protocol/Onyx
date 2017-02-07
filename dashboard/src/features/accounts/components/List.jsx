import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'account'

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Account Alias', 'Account ID']
    }
  }),
  BaseList.mapDispatchToProps(type)
)
