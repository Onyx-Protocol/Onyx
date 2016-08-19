import React from 'react';
import { Link } from 'react-router'

import PageNavigation from "./PageNavigation"
import PageHeader from "./PageHeader"
import SearchBar from "./SearchBar/SearchBar"

class ItemList extends React.Component {
  componentWillMount() {
    if (this.props.currentPage == -1) {
      this.props.getNextPage()
    }
  }

  render() {
    const label = this.props.label || this.props.type
    const title = label.charAt(0).toUpperCase() + label.slice(1) + "s"
    const keyProp = this.props.keyProp || "id"

    let header = [<PageHeader key='page-title' title={title} />]

    if (!this.props.skipCreate) { header.push (
      <button key='create-button' className="btn btn-primary btn-lg" onClick={this.props.showCreate}>
        Create new {label}
      </button>
    )}

    if (!this.props.skipQuery) { header.push(
      <SearchBar key='search-bar' submitQuery={this.props.submitQuery} queryString={this.props.query}/>
    )}

    if (this.props.pages[this.props.currentPage] !== undefined) {
      let pageNavigation = <PageNavigation
          currentPage={this.props.currentPage}
          lastPage={this.props.pages[this.props.currentPage].last_page}
          getPrevPage={this.props.getPrevPage}
          getNextPage={this.props.getNextPage} />

      return(
        <div>
          {header}
          {pageNavigation}

          {this.props.pages[this.props.currentPage].items.map((item) => {
            return <this.props.listItemComponent key={item[keyProp]} item={item} {...this.props.itemActions}/>
          })}

          {pageNavigation}
        </div>
       )
    } else {
      return(
        <div>
          {header}

          <div className="jumbotron text-center">
            <p>No results</p>
          </div>
        </div>
      )
    }
  }
}

export default ItemList
