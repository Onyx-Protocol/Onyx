import React from 'react'
import { BaseList, EmptyContent } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'unspent'
const title = <div className="emptyLabel">There are no unspent outputs on the blockchain</div>
const emptyListContent = <EmptyContent title={title} />

const newStateToProps = (state, ownProps) => ({
  ...BaseList.mapStateToProps(type, ListItem)(state, ownProps),
  skipCreate: true,
  label: 'Unspent outputs',
  emptyContent: emptyListContent
})

export default BaseList.connect(
  newStateToProps,
  BaseList.mapDispatchToProps(type)
)
