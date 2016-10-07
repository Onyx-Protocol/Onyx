import React from 'react'
import actions from 'actions'
import { connect as reduxConnect } from 'react-redux'
import { pluralize, humanize } from 'utility/string'
import { PageTitle, Pagination, SearchBar } from './'
import { pageSize } from 'utility/environment'

class ItemList extends React.Component {
  constructor(props) {
    super(props)

    this.state = { fetching: false }
  }

  componentWillMount() {
    this.fetchFirstPage(this.props)
  }

  componentWillReceiveProps(nextProps) {
    if (this.state.error) {
      if (this.props.searchState.queryString != nextProps.searchState.queryString) {
        this.setState({error: false})
      } else { return }
    }

    this.fetchFirstPage(nextProps)
  }

  fetchFirstPage(props) {
    if (props.items.length === 0 && !this.state.fetching) {
      this.setState({fetching: true})
      return this.props.fetchUntilPage(props.currentPage)
        .then((param) => {
          if (param && param.type == 'ERROR') {
            this.setState({error: true})
          }
        }).then(() => this.setState({fetching: false}))
    }
  }

  render() {
    const label = this.props.label || pluralize(humanize(this.props.type))

    let header = <div>
      <PageTitle
        title={label}
        actions={!this.props.skipCreate &&
          <button className='btn btn-link' onClick={this.props.showCreate}>
            New
          </button>}
      />

      {!this.props.skipQuery &&
        <SearchBar key='search-bar'
          updateQuery={this.props.updateQuery}
          {...this.props.searchState}
        />}
    </div>

    if (this.props.items.length > 0) {
      let pagination = <Pagination
          currentPage={this.props.currentPage}
          isLastPage={this.props.isLastPage}
          pushPage={this.props.pushPage} />

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

export const mapStateToProps = (type, itemComponent) => (state, ownProps) => {
  const currentPage = Math.max(parseInt(ownProps.location.query.page) || 1, 1)

  const currentIds = state[type].listView.itemIds
  const cursor = state[type].listView.cursor
  const lastPageIndex = Math.ceil(currentIds.length/pageSize) - 1
  const isLastPage = ((currentPage - 1) == lastPageIndex) && cursor && cursor.last_page
  const startIndex = (currentPage - 1) * pageSize
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

export const mapDispatchToProps = (type) => (dispatch) => {
  return {
    fetchUntilPage: (page) => dispatch(actions[type].fetchUntilPage(page)),
    pushPage: (page) => dispatch(actions[type].pushPage(page)),
    showCreate: () => dispatch(actions[type].showCreate),
    updateQuery: (query) => dispatch(actions[type].updateQuery(query))
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
