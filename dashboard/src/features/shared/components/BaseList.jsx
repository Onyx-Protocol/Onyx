import { connect as reduxConnect } from 'react-redux'
import actions from 'actions'
import ItemList from 'components/ItemList'
import { pageSize } from 'utility/environment'

export const mapStateToProps = (type, itemComponent) => (state) => {
  const currentIds = state[type].listView.itemIds
  const currentPage = state[type].listView.pageIndex
  const cursor = state[type].listView.cursor
  const lastPageIndex = Math.ceil(currentIds.length/pageSize) - 1

  const isLastPage = (currentPage == lastPageIndex) && cursor && cursor.last_page

  const startIndex = currentPage * pageSize
  const items = currentIds.slice(startIndex, startIndex + pageSize).map(
    id => state[type].items[id]
  )

  return {
    items: items,
    currentPage: currentPage,
    isLastPage: isLastPage,
    type: type,
    listItemComponent: itemComponent,
    searchState: {
      queryString: state[type].listView.query,
      queryTime: state[type].listView.queryTime,
    },
  }
}

export const mapDispatchToProps = (type) => (dispatch) => ({
  incrementPage: () => dispatch(actions[type].incrementPage()),
  decrementPage: () => dispatch(actions[type].decrementPage()),
  showCreate: () => dispatch(actions[type].showCreate),
  updateQuery: (query) => dispatch(actions[type].updateQuery(query))
})

export const connect = (state, dispatch, component = ItemList) => reduxConnect(
  state,
  dispatch
)(component)

export default {
  mapStateToProps,
  mapDispatchToProps,
  connect,
}
