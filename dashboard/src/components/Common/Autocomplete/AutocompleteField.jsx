import React from 'react'
import styles from './AutocompleteField.scss'
import Autosuggest from 'react-autosuggest'
import actions from 'actions'

class AutocompleteField extends React.Component {
  constructor() {
    super()

    this.state = {
      suggestions: []
    }

    this.getSuggestionValue = this.getSuggestionValue.bind(this)
    this.renderSuggestion = this.renderSuggestion.bind(this)
    this.onSuggestionsFetchRequested = this.onSuggestionsFetchRequested.bind(this)
    this.onSuggestionsClearRequested = this.onSuggestionsClearRequested.bind(this)
    this.tabCheck = this.tabCheck.bind(this)
  }

  getSuggestions(value) {
    const escapedValue = (value.trim()).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

    if (escapedValue === '') {
      return []
    }

    const regex = new RegExp(escapedValue, 'i')

    return this.props.items.filter(item => regex.test(item.alias))
  }


  getSuggestionValue(suggestion) {
    return suggestion.alias
  }

  renderSuggestion(suggestion) {
    return (
      <div onMouseOver={this.handleHover.bind(this, suggestion.alias)}>
        <span>{suggestion.alias}</span>
      </div>
    )
  }

  onSuggestionsFetchRequested({ value }) {
    if (this.props.autocompleteIsLoaded) {
      this.setState({suggestions: this.getSuggestions(value)})
    } else {
      this.props.fetchAll().then(() => {
        this.setState({suggestions: this.getSuggestions(value)})
        this.props.didLoadAutocomplete()
      })
    }
  }

  onSuggestionsClearRequested() {
    this.setState({
      suggestions: []
    })
  }

  handleHover(suggestion) {
    this.props.fieldProps.onChange(suggestion)
  }

  tabCheck(event, suggestions, fieldProps) {
    if (event.keyCode == 9) {
      const suggestion = suggestions[0]["alias"]
      const input = fieldProps.value.toLowerCase()
      if (suggestion.toLowerCase().startsWith(input)) {
        fieldProps.onChange(suggestion)
      }
    }
  }

  render() {
    const { suggestions } = this.state
    const { fieldProps } = this.props

    return (
      <Autosuggest
        theme={styles}
        suggestions={suggestions}
        onSuggestionsFetchRequested={this.onSuggestionsFetchRequested}
        onSuggestionsClearRequested={this.onSuggestionsClearRequested}
        onSuggestionSelected={(event) => event.preventDefault()}
        getSuggestionValue={this.getSuggestionValue}
        renderSuggestion={this.renderSuggestion}
        inputProps={{
          className: `form-control ${this.props.className}`,
          value: fieldProps.value,
          placeholder: this.props.placeholder,
          onChange: (event, { newValue }) => fieldProps.onChange(newValue),
          onKeyDown: (event) => this.tabCheck(event, suggestions, fieldProps)}}
      />
    )
  }
}

export default AutocompleteField

export const mapStateToProps = (type) => (state) => ({
  autocompleteIsLoaded: state[type].autocompleteIsLoaded,
  items: Object.keys(state[type].items).map(k => state[type].items[k])
})

export const mapDispatchToProps = (type) => (dispatch) => ({
  didLoadAutocomplete: () => dispatch(actions[type].didLoadAutocomplete),
  fetchAll: () => dispatch(actions[type].fetchAll())
})
