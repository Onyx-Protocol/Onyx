import React from 'react'
import { BaseList } from 'features/shared/components'
import ListItem from './ListItem/ListItem'
import actions from 'actions'

const type = 'transaction'

let EmptyContent = <div>
  <p>
    To build your first transaction, you will need to:
  </p>
  <ol>
    <li><a href="/accounts/create">create an account</a></li>
    <li><a href="/assets/create">create an asset</a></li>
  </ol>
  <br />
  <p>
    Learn more about how to build transactions and check balances by checking
    out the <a href="/docs/core/build-applications/transaction-basics" target="_blank">Transactions</a> guide
    in the documentation.
  </p>
</div>

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
    emptyContent: EmptyContent,
    showWelcome: true
  }),
  (dispatch) => ({
    ...BaseList.mapDispatchToProps(type)(dispatch),
    getLatest: (query) => dispatch(actions.transaction.fetchPage(query, 1, { refresh: true })),
  }),
  List
)
