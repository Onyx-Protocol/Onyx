import React from 'react'
import styles from './SearchBar.scss'

class SearchBar extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      query: this.props.queryString || '',
      sumBy: this.props.sumBy || ''
    }
    this.state.showClear = this.state.query != '' || this.state.sumBy != ''

    this.filterKeydown = this.filterKeydown.bind(this)
    this.filterOnChange = this.filterOnChange.bind(this)
    this.sumByOnChange = this.sumByOnChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
    this.clearQuery = this.clearQuery.bind(this)
  }

  componentWillReceiveProps(nextProps) {
    // Override text field with default query when provided
    this.setState({query: nextProps.queryString})
  }

  filterKeydown(event) {
    this.setState({lastKeypress: event.key})
  }

  filterOnChange(event) {
    const input = event.target
    const key = this.state.lastKeypress
    let value = event.target.value
    let cursorPosition = input.selectionStart

    switch (key) {
      case '=':
        value = value.substr(0, cursorPosition)
          + "''"
          + value.substr(cursorPosition)

        cursorPosition += 1
        break
      case '"':
        value = value.substr(0, value.length - 1) + "'"
        break
      case "'":
        if (value[cursorPosition] == "'" &&
            value[cursorPosition - 1] == "'") {
          value = value.substr(0, cursorPosition-1)
            + value.substr(cursorPosition)
        }
        break
      case '(':
        value = value.substr(0, cursorPosition)
          + ')'
          + value.substr(cursorPosition)

        break
      case ')':
        if (value[cursorPosition] == ')' &&
            value[cursorPosition - 1] == ')') {
          value = value.substr(0, cursorPosition-1)
            + value.substr(cursorPosition)
        }
        break
    }

    this.setState({query: value})

    // Setting selection range only works after the onChange
    // handler has completed
    setTimeout(() => {
      input.setSelectionRange(cursorPosition, cursorPosition)
    }, 0)
  }

  sumByOnChange(event) {
    this.setState({sumBy: event.target.value})
  }

  handleSubmit(event) {
    event.preventDefault()

    if (this.state.query == '' && this.state.sumBy == '') {
      if (this.props.queryString || this.props.sumBy) {
        this.clearQuery()
      }
      return
    }

    this.setState({ showClear: true })

    const query = {}
    if (this.state.query) query.filter = this.state.query
    if (this.state.sumBy) query.sum_by = this.state.sumBy

    this.props.pushList(query)
  }

  clearQuery() {
    this.setState({ query: '', sumBy: '', showClear: false })
    this.props.pushList()
  }

  render() {
    let showSumBy = false
    let searchFieldClass = styles.search_field_full

    if (this.props.sumBy !== undefined) {
      showSumBy = true
      searchFieldClass = styles.search_field_half
    }

    return (
      <div className={styles.search_bar}>
        <form onSubmit={this.handleSubmit}>
          <span className={searchFieldClass}>
            <label>Filter</label>
            <input
              value={this.state.query}
              onKeyDown={this.filterKeydown}
              onChange={this.filterOnChange}
              className={`form-control ${styles.search_input}`}
              type='search'
              autoFocus='autofocus'
              placeholder='Enter predicate...' />
          </span>

          {showSumBy &&
            <span className={styles.sum_by_field}>
              <label>Sum By</label>
              <input
                value={this.state.sumBy}
                onChange={this.sumByOnChange}
                className={`form-control ${styles.search_input}`}
                type='search'
                placeholder='asset_alias, asset_id' />
            </span>}

          <div className={styles.search_button_container}>
            <button type='submit' className={`btn btn-primary ${styles.search_button}`} >
              Filter
            </button>

            {this.state.showClear &&
              <button type='button'
                className={`close ${styles.clear_search}`}
                onClick={this.clearQuery}>
                  Reset
              </button>}
          </div>
        </form>

        {this.state.showClear && <span className={styles.queryTime}>
          Queried at {this.props.queryTime}
        </span>}
      </div>
    )
  }
}

export default SearchBar
