import React from 'react'
import actions from 'actions'
import { connect as reduxConnect } from 'react-redux'
import { pluralize, capitalize, humanize } from 'utility/string'
import { PageContent, PageTitle, Pagination, SearchBar } from '../'
import EmptyList from './EmptyList'
import { pageSize } from 'utility/environment'

class ItemList extends React.Component {
  render() {
    const label = this.props.label || pluralize(humanize(this.props.type))
    const objectName = label.slice(0,-1)
    const newLabel = 'New ' + objectName
    const actions = [...(this.props.actions || [])]
    const newButton = <button key='showCreate' className='btn btn-primary' onClick={this.props.showCreate}>
      + {newLabel}
    </button>
    if (!this.props.skipCreate && !this.props.showFirstTimeFlow) {
      actions.push(newButton)
    }

    let header = <div>
      <PageTitle
        title={capitalize(label)}
        actions={actions}
      />

      {!this.props.skipQuery && !this.props.showFirstTimeFlow &&
        <SearchBar key='search-bar'
          {...this.props.searchState}
          pushList={this.props.pushList}
          currentFilter={this.props.currentFilter}
          defaultFilter={this.props.defaultFilter}
        />}
    </div>

    if (this.props.noResults) {
      return(
        <div className='flex-container'>
          {header}

          <EmptyList
            firstTimeContent={this.props.firstTimeContent}
            type={this.props.type}
            objectName={objectName}
            newButton={newButton}
            showFirstTimeFlow={this.props.showFirstTimeFlow}
            skipCreate={this.props.skipCreate}
            loadedOnce={this.props.loadedOnce}
            currentFilter={this.props.currentFilter} />

        </div>
      )
    } else {
      let pagination = <Pagination
          currentPage={this.props.currentPage}
          currentFilter={this.props.currentFilter}
          isLastPage={this.props.isLastPage}
          pushList={this.props.pushList} />

      const items = this.props.items.map((item) =>
        <this.props.listItemComponent key={item.id} item={item} {...this.props.itemActions}/>)
      const Wrapper = this.props.wrapperComponent

      return(
        <div className='flex-container'>
          {header}

          <PageContent>
            {Wrapper ? <Wrapper {...this.props.wrapperProps}>{items}</Wrapper> : items}

            {pagination}
          </PageContent>
        </div>
      )
    }
  }
}

export const mapStateToProps = (type, itemComponent, additionalProps = {}) => (state, ownProps) => {
  const currentPage = Math.max(parseInt(ownProps.location.query.page) || 1, 1)
  // TODO: this should be renamed `currentQuery`; we should
  // do some renaminng in here
  const currentFilter = ownProps.location.query || {}
  const currentQueryString = currentFilter.filter || ''
  const currentQuery = state[type].queries[currentQueryString] || {}
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

    loadedOnce: Object.keys(state[type].queries).length > 0,
    type: type,
    listItemComponent: itemComponent,
    searchState: { queryTime: currentQuery.queryTime },

    noResults: items.length == 0,
    showFirstTimeFlow: items.length == 0 && currentQueryString == '',

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
  ItemList,
}
