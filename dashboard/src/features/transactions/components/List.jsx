import React from 'react'
import { BaseList, EmptyContent } from 'features/shared/components'
import { Link } from 'react-router'
import ListItem from './ListItem/ListItem'
import actions from 'actions'

const type = 'transaction'

const firstTimeContent = <EmptyContent title="Welcome to Chain Core!">
  To build your first transaction, you will need to:
  <ol>
    <li>
      <Link to='/accounts/create'>
      create an account
      </Link>
    </li>
    <li>
      <Link to='/assets/create'>
      create an asset
      </Link>
    </li>
  </ol>
  <a href="/docs/core/build-applications/transaction-basics" target="_blank">
    Learn more
  </a> about how to build, sign, and submit transactions.
</EmptyContent>

class List extends React.Component {
  componentWillReceiveProps(nextProps) {
    if (nextProps.blockHeight != this.props.blockHeight) {
      if (nextProps.currentPage == 1) {
        this.props.getLatest(nextProps.currentFilter)
      }
    }
  }

  render() {
    const ItemList = BaseList.ItemList
    return (<ItemList {...this.props} />)
  }
}

export default BaseList.connect(
  (state, ownProps) => ({
    ...BaseList.mapStateToProps(type, ListItem)(state, ownProps),
    blockHeight: state.core.blockHeight,
    firstTimeContent
  }),
  (dispatch) => ({
    ...BaseList.mapDispatchToProps(type)(dispatch),
    getLatest: (query) => dispatch(actions.transaction.fetchPage(query, 1, { refresh: true })),
  }),
  List
)
