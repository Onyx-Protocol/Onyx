import React from 'react'
import styles from './XpubField.scss'
import { SelectField, FieldLabel, HiddenField } from 'features/shared/components'
import { TextField } from '../'
import { connect } from 'react-redux'
import { actions } from 'features/mockhsm'

const methodOptions = {
  mockhsm: 'Use Mock HSM key',
  provide: 'Provide existing Xpub',
  generate: 'Generate Mock HSM key',
}

class XpubField extends React.Component {
  constructor(props) {
    super(props)

    this.state = { selectedType: Object.keys(methodOptions)[0] }
  }

  componentDidMount() {
    if (!this.props.autocompleteIsLoaded) {
      this.props.fetchAll().then(() => {
        this.props.didLoadAutocomplete()
      })
    }
  }

  render() {
    const radioChanged = event => {
      this.setState({ selectedType: event.target.value })
    }

    return (
      <div className={styles.main}>
        <FieldLabel>Key {this.props.index + 1}</FieldLabel>

        <div className={styles.options}>
          {Object.keys(methodOptions).map((key) =>
            <label key={`key-${this.props.index}-option-${key}`}>
              <input type='radio'
                name={`keys-${this.props.index}`}
                value={key}
                checked={key == this.state.selectedType}
                onChange={radioChanged}
              />
              {methodOptions[key]}
            </label>
          )}
        </div>

        {this.state.selectedType == 'mockhsm' &&
          <SelectField options={this.props.mockhsmKeys}
            valueKey='xpub'
            labelKey='label'
            fieldProps={this.props.fieldProps} />
        }

        {this.state.selectedType == 'provide' &&
          <TextField key={this.props.index} fieldProps={this.props.fieldProps} />}

        {this.state.selectedType == 'generate' &&
          <HiddenField key={this.props.index} fieldProps={this.props.fieldProps} />}
      </div>
    )
  }
}

XpubField.propTypes = {
  index: React.PropTypes.number,
  fieldProps: React.PropTypes.object,
  mockhsmKeys: React.PropTypes.array,
  autocompleteIsLoaded: React.PropTypes.bool,
  fetchAll: React.PropTypes.func,
  didLoadAutocomplete: React.PropTypes.func,
}

export default connect(
  (state) => {
    let keys = []
    for (var key in state.mockhsm.items) {
      const item = state.mockhsm.items[key]
      keys.push({
        ...item,
        label: item.alias ? item.alias : item.id.slice(0, 32) + '...'
      })
    }

    return {
      autocompleteIsLoaded: state.mockhsm.autocompleteIsLoaded,
      mockhsmKeys: keys,
    }
  },
  (dispatch) => ({
    didLoadAutocomplete: () => dispatch(actions.didLoadAutocomplete),
    fetchAll: (cb) => dispatch(actions.fetchAll(cb)),
  })
)(XpubField)
