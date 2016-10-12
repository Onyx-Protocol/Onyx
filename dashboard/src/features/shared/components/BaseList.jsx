import React from 'react'
import actions from 'actions'
import { connect as reduxConnect } from 'react-redux'
import { pluralize, humanize } from 'utility/string'
import { PageTitle, Pagination, SearchBar } from './'
import { pageSize } from 'utility/environment'

export class ItemList extends React.Component {
  render() {
    const label = this.props.label || pluralize(humanize(this.props.type))

    const actions = [...(this.props.actions || [])]
    if (!this.props.skipCreate) {
      actions.push(<button key='showCreate' className='btn btn-link' onClick={this.props.showCreate}>
        + New
      </button>)
    }

    let header = <div>
      <PageTitle
        title={label}
        actions={actions}
      />

      {!this.props.skipQuery &&
        <SearchBar key='search-bar'
          {...this.props.searchState}
          pushList={this.props.pushList}
          queryString={this.props.currentFilter}
        />}
    </div>

    if (this.props.items.length > 0) {
      let pagination = <Pagination
          currentPage={this.props.currentPage}
          currentFilter={this.props.currentFilter}
          isLastPage={this.props.isLastPage}
          pushList={this.props.pushList} />

      return(
        <div>
          {header}

          {this.props.children}

          {this.props.items.map((item) =>
            <this.props.listItemComponent key={item.id} item={item} {...this.props.itemActions}/>
          )}

          {pagination}
        </div>
       )
    } else {
      return(
        <div>
          {header}

          {this.props.children}

          <div className='jumbotron text-center'>
            <p>No results</p>
          </div>
        </div>
      )
    }
  }
}

export const mapStateToProps = (type, itemComponent, additionalProps = {}) => (state, ownProps) => {
  const currentPage = Math.max(parseInt(ownProps.location.query.page) || 1, 1)
  const currentFilter = ownProps.location.query.filter || ''
  const currentQuery = state[type].queries[currentFilter] || {}
  const currentIds = currentQuery.itemIds || []
  const cursor = currentQuery.cursor || {}

  const lastPageIndex = Math.ceil(currentIds.length/pageSize) - 1
  const isLastPage = ((currentPage - 1) == lastPageIndex) && cursor && cursor.last_page
  const startIndex = (currentPage - 1) * pageSize
  const items = currentIds.slice(startIndex, startIndex + pageSize).map(
    id => state[type].items[id]
  ).filter(item => item != undefined)

  return {
    currentPage: currentPage,
    currentFilter: currentFilter,
    items: items,
    isLastPage: isLastPage,

    type: type,
    listItemComponent: itemComponent,
    searchState: {
      // queryString: state[type].listView.query,
      queryTime: currentQuery.queryTime,
    },
    ...additionalProps
  }
}

export const mapDispatchToProps = (type) => (dispatch) => {
  return {
    pushList: (query, pageNumber) => dispatch(actions[type].pushList(query, pageNumber)),
    showCreate: () => dispatch(actions[type].showCreate),
  }
}

export const connect = (state, dispatch, component = ItemList) => reduxConnect(
  state,
  dispatch
)(component)

export default {
  mapStateToProps,
  mapDispatchToProps,
  connect,
}
