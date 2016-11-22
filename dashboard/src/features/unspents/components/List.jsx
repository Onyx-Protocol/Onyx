import React from 'react'
import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'unspent'

const EmptyContent = <div className="emptyContainer">
  <p>
    You do not have any unspent outputs.
  </p>
  <p>
    An output is considered unspent when it has not yet been used as an input
    to a new transaction. All asset units on a blockchain exist in the unspent output set.
  </p>
  <p>
    Learn more about the basic functions of unspent outputs in the&nbsp;
    <a href="/docs/core/build-applications/accounts" target="_blank">Unspent Outputs</a> guide
    of the documentation.
  </p>
</div>

const newStateToProps = (state, ownProps) => ({
  ...BaseList.mapStateToProps(type, ListItem)(state, ownProps),
  skipCreate: true,
  label: 'Unspent outputs',
  emptyContent: EmptyContent
})

export default BaseList.connect(
  newStateToProps,
  BaseList.mapDispatchToProps(type)
)
