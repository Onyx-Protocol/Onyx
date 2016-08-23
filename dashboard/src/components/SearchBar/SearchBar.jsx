import React from 'react';
import styles from "./SearchBar.scss"

class SearchBar extends React.Component {
  constructor(props) {
    super(props)
    this.state = { query: this.props.queryString || "" }
    this.state.showClear = this.state.query != ""

    this.handleChange = this.handleChange.bind(this)
    this.handleSubmit = this.handleSubmit.bind(this)
    this.clearQuery = this.clearQuery.bind(this)
  }

  componentWillReceiveProps(nextProps) {
    // Override text field with default query when provided
    this.setState({query: nextProps.queryString})
  }

  handleChange(event) {
    this.setState({
      query: this.refs.queryField.value
    })
  }

  handleSubmit(event) {
    event.preventDefault()

    this.setState({ showClear: true })
    this.props.submitQuery(this.state.query)
  }

  clearQuery(event) {
    this.setState({ query: "", showClear: false })
    this.props.submitQuery("")
  }

  render() {
    let clearButton = this.state.showClear ? <button type="button"
                   className={`close ${styles.clear_search}`}
                   onClick={this.clearQuery}>
                     &times;
                 </button> : ""
    return (
      <div className={styles.search_bar}>
        <form onSubmit={this.handleSubmit}>
          <span className={styles.search_field}>
            <input ref="queryField"
                   value={this.state.query}
                   onChange={this.handleChange}
                   className={`form-control ${styles.search_input}`}
                   type="search"
                   autoFocus="autofocus"
                   placeholder="ChQL query..." />

            {clearButton}
          </span>
          <button type="submit" className={`btn btn-primary ${styles.search_button}`} >
            Search
          </button>
        </form>
      </div>
    )
  }
}

export default SearchBar
