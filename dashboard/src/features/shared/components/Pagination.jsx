import React from 'react'

class Pagination extends React.Component {
  render() {
    let prevClass = 'btn btn-default ' + (this.props.currentPage > 1 ? '' : 'disabled')
    let nextClass = 'btn btn-default ' + (this.props.isLastPage ? 'disabled' : '')

    const nextPage = () => this.props.pushList(this.props.currentFilter, this.props.currentPage + 1)
    const prevPage = () => this.props.pushList(this.props.currentFilter, this.props.currentPage - 1)

    return (
      <ul className='pager'>
        <li className='previous'>
          <a className={prevClass} onClick={prevPage}>
            &larr; Prev
          </a>
        </li>
        <li>Page {this.props.currentPage}</li>
        <li className='next'>
          <a className={nextClass} onClick={nextPage}>
            Next &rarr;
          </a>
        </li>
      </ul>
    )
  }
}

export default Pagination
