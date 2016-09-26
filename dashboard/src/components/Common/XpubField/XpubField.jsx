import React from 'react'
import { SelectField, TextField } from '../'
import styles from './XpubField.scss'

const methodOptions = {
  mockhsm: 'Use Mock HSM key',
  provide: 'Provide existing Xpub'
}

class XpubField extends React.Component {
  constructor(props) {
    super(props)

    this.state = { selectedType: Object.keys(methodOptions)[0] }
  }

  render() {
    const radioChanged = event => {
      this.setState({ selectedType: event.target.value })
    }

    const keys = this.props.mockhsmKeys.map(item => ({
      ...item,
      label: item.alias ? item.alias : item.id.slice(0, 32) + '...'
    }))

    return (
      <div className={styles.main}>
        <label>Key {this.props.index + 1}</label>

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
          <SelectField options={keys}
            valueKey='xpub'
            labelKey='label'
            fieldProps={this.props.fieldProps} />
        }

        {this.state.selectedType == 'provide' &&
          <TextField key={this.props.index} fieldProps={this.props.fieldProps} />}
      </div>
    )
  }
}

export default XpubField
