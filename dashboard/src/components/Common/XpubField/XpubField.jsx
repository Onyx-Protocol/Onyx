import React from 'react'
import styles from './XpubField.scss'
import { SelectField, FieldLabel } from 'features/shared/components'
import { TextField } from '../'
import { connect } from 'react-redux'
import { actions } from 'features/mockhsm'

const methodOptions = {
  generate: 'Generate new Mock HSM key',
  mockhsm: 'Use existing Mock HSM key',
  provide: 'Provide existing Xpub',
}

class XpubField extends React.Component {
  constructor(props) {
    super(props)

    this.state = {
      generate: '',
      mockhsm: '',
      provide: ''
    }
  }

  componentDidMount() {
    if (!this.props.autocompleteIsLoaded) {
      this.props.fetchAll().then(() => {
        this.props.didLoadAutocomplete()
      })
    }

    this.props.typeProps.onChange(Object.keys(methodOptions)[0])
  }

  render() {
    const {
      typeProps,
      valueProps,
      mockhsmKeys,
    } = this.props

    const typeOnChange = event => {
      const value = typeProps.onChange(event).value

      typeProps.onChange(value)
      valueProps.onChange(this.state[value] || '')
    }

    const valueOnChange = event => {
      const value = valueProps.onChange(event).value
      this.setState({ [typeProps.value]: value })
    }

    return (
      <div className={styles.main}>
        <FieldLabel>Key {this.props.index + 1}</FieldLabel>

        <div className={styles.options}>
          {Object.keys(methodOptions).map((key) =>
            <label key={`key-${this.props.index}-option-${key}`}>
              <input type='radio'
                name={`keys-${this.props.index}`}
                onChange={typeOnChange}
                checked={key == typeProps.value}
                value={key}
              />
              {methodOptions[key]}
            </label>
          )}
        </div>

        {typeProps.value == 'mockhsm' &&
          <SelectField options={mockhsmKeys}
            valueKey='xpub'
            labelKey='label'
            fieldProps={{...valueProps, onChange: valueOnChange}} />
        }

        {typeProps.value == 'provide' &&
          <TextField
            fieldProps={{...valueProps, onChange: valueOnChange}}
            placeholder='Enter Xpub' />}

        {typeProps.value == 'generate' &&
          <TextField
            fieldProps={{...valueProps, onChange: valueOnChange}}
            placeholder='Alias for generated key (leave blank for automatic value)' />}
      </div>
    )
  }
}

XpubField.propTypes = {
  index: React.PropTypes.number,
  typeProps: React.PropTypes.object,
  valueProps: React.PropTypes.object,
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
