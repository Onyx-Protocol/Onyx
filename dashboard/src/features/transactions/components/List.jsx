import React from 'react'
import { BaseList } from 'features/shared/components'
import ListItem from './ListItem/ListItem'
import actions from 'actions'

const type = 'transaction'

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
    blockHeight: state.core.blockHeight
  }),
  (dispatch) => ({
    ...BaseList.mapDispatchToProps(type)(dispatch),
    getLatest: (query) => dispatch(actions.transaction.fetchPage(query, 1, { refresh: true })),
  }),
  List
)
