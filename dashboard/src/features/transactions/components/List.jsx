import React from 'react'
import { BaseList } from 'features/shared/components'
import ListItem from './ListItem/ListItem'
import actions from 'actions'

const type = 'transaction'

const emptyListContent = <div>
  <h2>Welcome to Chain Core!</h2>
  <div className="emptyContainer">
    <p>
      To build your first transaction, you will need to:
    </p>
    <ol className="emptyList">
      <li className="emptyListItem"><a href="/accounts/create">create an account</a></li>
      <li className="emptyListItem"><a href="/assets/create">create an asset</a></li>
    </ol>
    <br />
    Learn more about how to build, sign, and submit&nbsp;
    <a href="/docs/core/build-applications/transaction-basics" target="_blank">
      transactions
    </a>.
  </div>
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
    emptyContent: emptyListContent
  }),
  (dispatch) => ({
    ...BaseList.mapDispatchToProps(type)(dispatch),
    getLatest: (query) => dispatch(actions.transaction.fetchPage(query, 1, { refresh: true })),
  }),
  List
)
