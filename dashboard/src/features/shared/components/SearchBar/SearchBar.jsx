import React from 'react'
import styles from './SearchBar.scss'

class SearchBar extends React.Component {
  constructor(props) {
    super(props)

    // TODO: examine renaming and refactoring for clarity. Consider moving
    // away from local state if possible.
    this.state = {
      query: this.props.currentFilter.filter || '',
      sumBy: this.props.currentFilter.sum_by || '',
      sumByVisible: false,
    }
    this.state.showClear = (this.state.query != (this.props.defaultFilter || '')) || this.state.sumBy != ''
    this.state.sumByVisible = this.state.sumBy != ''

    this.filterKeydown = this.filterKeydown.bind(this)
    this.filterOnChange = this.filterOnChange.bind(this)
    this.sumByOnChange = this.sumByOnChange.bind(this)
    this.showSumBy = this.showSumBy.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
    this.clearQuery = this.clearQuery.bind(this)
  }

  componentWillReceiveProps(nextProps) {
    // Override text field with default query when provided
    if (nextProps.currentFilter.filter != this.props.currentFilter.filter) {
      this.setState({query: nextProps.currentFilter.filter})
    }
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

  showSumBy() {
    this.setState({sumByVisible: true})
  }

  sumByOnChange(event) {
    this.setState({sumBy: event.target.value})
  }

  handleSubmit(event) {
    event.preventDefault()

    const query = {}
    const state = {
      showClear: (this.state.query && (this.state.query != this.props.defaultFilter)) || this.state.sumBy
    }

    if (this.state.query) {
      query.filter = this.state.query
    } else if (this.props.defaultFilter) {
      state.query = this.props.defaultFilter
      query.filter = this.props.defaultFilter
    }
    if (this.state.sumBy) query.sum_by = this.state.sumBy

    this.setState(state)
    this.props.pushList(query)
  }

  clearQuery() {
    const newState = { query: (this.props.defaultFilter || ''), sumBy: '', showClear: false}
    this.setState(newState)

    const query = {}
    if (newState.query) { query.filter = newState.query }
    this.props.pushList(query)
  }

  render() {
    let usesSumBy = false
    let searchFieldClass = styles.search_field_full

    if (this.props.sumBy !== undefined) usesSumBy = true
    if (this.state.sumByVisible) searchFieldClass = styles.search_field_half

    return (
      <div className={styles.main}>
        <form onSubmit={this.handleSubmit}>
          <span className={`${styles.searchField} ${searchFieldClass}`}>
            <input
              value={this.state.query || ''}
              onKeyDown={this.filterKeydown}
              onChange={this.filterOnChange}
              className={`form-control ${styles.search_input}`}
              type='search'
              autoFocus='autofocus'
              placeholder='Enter filter...' />

            {usesSumBy && !this.state.sumByVisible &&
              <span onClick={this.showSumBy} className={styles.showSumBy}>set sum_by</span>}
          </span>

          {usesSumBy && this.state.sumByVisible &&
            <span className={styles.sum_by_field}>
              <input
                value={this.state.sumBy}
                onChange={this.sumByOnChange}
                className={`form-control ${styles.search_input} ${styles.sum_by_input}`}
                type='search'
                autoFocus='autofocus'
                placeholder='Enter sum_by...' />
            </span>}

            {/* This is required for form submission */}
            <input type='submit' className={styles.submit} tabIndex='-1' />
        </form>

        <span className={styles.queryTime}>
          {/* TODO: in the future there may be objects with default filters that
              do not require a filter; this is a stopgap measure for balances. */}
          {this.props.defaultFilter && !this.state.query.trim() && 'Filter is required • '}

          Queried at {this.props.queryTime}

          {this.state.showClear && <span>
            {' • '}
            <span type='button'
              className={styles.clearSearch}
              onClick={this.clearQuery}>
                {this.props.defaultFilter ? 'Restore default filter' : 'Clear filter'}
            </span>
          </span>}
        </span>
      </div>
    )
  }
}

export default SearchBar
