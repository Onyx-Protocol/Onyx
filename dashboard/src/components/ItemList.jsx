import React from 'react'
import { pluralize, capitalize } from '../utility/string'

import Pagination from './Pagination'
import PageHeader from './PageHeader/PageHeader'
import SearchBar from './SearchBar/SearchBar'

class ItemList extends React.Component {
  constructor(props) {
    super(props)

    this.state = { mounted: false }
  }

  componentWillMount() {
    if (this.props.items.length === 0) {
      Promise.resolve(this.fetchFirstPage(this.props)).then(() => {
        this.setState({mounted: true})
      })
    } else {
      this.setState({mounted: true})
    }
  }

  componentWillReceiveProps(nextProps) {
    if (this.state.mounted) {
      this.fetchFirstPage(nextProps)
    }
  }

  fetchFirstPage(props) {
    if (props.items.length === 0) {
      return this.props.incrementPage()
    }
  }

  render() {
    const label = this.props.label || pluralize(capitalize(this.props.type))

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

          <div className='jumbotron text-center'>
            <p>No results</p>
          </div>
        </div>
      )
    }
  }
}

export default ItemList
