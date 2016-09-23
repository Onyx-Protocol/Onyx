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

    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
    this.clearQuery = this.clearQuery.bind(this)
  }

  componentWillReceiveProps(nextProps) {
    // Override text field with default query when provided
    this.setState({query: nextProps.queryString})
  }

  handleChange() {
    let newState = {
      query: this.refs.queryField.value
    }
    if (this.refs.sumByField) {
      newState.sumBy = this.refs.sumByField.value
    }
    this.setState(newState)
  }

  handleSubmit(event) {
    event.preventDefault()

    this.setState({ showClear: true })
    this.props.updateQuery({
      query: this.state.query,
      sumBy: this.state.sumBy
    })
  }

  clearQuery() {
    this.setState({ query: '', sumBy: '', showClear: false })
    this.props.updateQuery('')
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
            <input ref='queryField'
                   value={this.state.query}
                   onChange={this.handleChange}
                   className={`form-control ${styles.search_input}`}
                   type='search'
                   autoFocus='autofocus'
                   placeholder='Enter predicate...' />
          </span>

          {showSumBy &&
            <span className={styles.sum_by_field}>
              <label>Sum By</label>
              <input ref='sumByField'
                value={this.state.sumBy}
                onChange={this.handleChange}
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
      </div>
    )
  }
}

export default SearchBar
