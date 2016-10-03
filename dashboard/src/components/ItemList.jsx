import React from 'react'
import { pluralize, humanize } from '../utility/string'

import Pagination from './Pagination'
import PageHeader from './PageHeader/PageHeader'
import SearchBar from './SearchBar/SearchBar'

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
      return this.props.incrementPage()
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
      <PageHeader key='page-title'
        title={label}
        buttonAction={this.props.showCreate}
        buttonLabel='New'
        showActionButton={!this.props.skipCreate}
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
          incrementPage={this.props.incrementPage}
          decrementPage={this.props.decrementPage} />

      return(
        <div>
          {header}

          {this.props.children}

          {pagination}

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

export default ItemList
