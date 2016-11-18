import React from 'react'
import styles from './XpubField.scss'
import { SelectField, FieldLabel } from '../'
import { TextField } from 'components/Common'
import { connect } from 'react-redux'
import actions from 'features/mockhsm/actions'

const methodOptions = {
  generate: 'Generate new MockHSM key',
  mockhsm: 'Use existing MockHSM key',
  provide: 'Provide existing xpub',
}

class XpubField extends React.Component {
  constructor(props) {
    super(props)

    this.state = {
      generate: '',
      mockhsm: '',
      provide: '',
      autofocusInput: false
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
      valueProps.onChange(this.state[value] || '')
      this.setState({ autofocusInput: true })
    }

    const valueOnChange = event => {
      const value = valueProps.onChange(event).value
      this.setState({ [typeProps.value]: value })
    }

    const fields = {
      'mockhsm': <SelectField options={mockhsmKeys}
        autoFocus={this.state.autofocusInput}
        valueKey='xpub'
        labelKey='label'
        fieldProps={{...valueProps, onChange: valueOnChange}} />,
      'provide': <TextField
        autoFocus={this.state.autofocusInput}
        fieldProps={{...valueProps, onChange: valueOnChange}}
        placeholder='Enter xpub' />,
      'generate': <TextField
        autoFocus={this.state.autofocusInput}
        fieldProps={{...valueProps, onChange: valueOnChange}}
        placeholder='Alias for generated key (leave blank for automatic value)' />,
    }

    return (
      <div className={styles.main}>
        <FieldLabel>Key {this.props.index + 1}</FieldLabel>

        <table className={styles.options}>
          {Object.keys(methodOptions).map((key) =>
            <tr>
              <td className={styles.label}>
                <label key={`key-${this.props.index}-option-${key}`}>
                  <input type='radio'
                    className={styles.radio}
                    name={`keys-${this.props.index}`}
                    onChange={typeOnChange}
                    checked={key == typeProps.value}
                    value={key}
                  />
                  {methodOptions[key]}
                </label>
              </td>

              <td className={styles.field}>
                {typeProps.value == key && fields[key]}
              </td>
            </tr>
          )}
        </table>
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
