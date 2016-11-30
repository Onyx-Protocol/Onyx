import React from 'react'
import { BaseList, EmptyContent } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'unspent'
const firstTimeContent = <EmptyContent
  title="There are no unspent outputs on the blockchain" />

const newStateToProps = (state, ownProps) => ({
  ...BaseList.mapStateToProps(type, ListItem)(state, ownProps),
  skipCreate: true,
  label: 'Unspent outputs',
  firstTimeContent
})

export default BaseList.connect(
  newStateToProps,
  BaseList.mapDispatchToProps(type)
)
