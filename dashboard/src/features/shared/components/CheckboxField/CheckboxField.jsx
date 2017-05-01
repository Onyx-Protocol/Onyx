import React from 'react'
import pick from 'lodash/pick'
import styles from './CheckboxField.scss'

const CHECKBOX_FIELD_PROPS = [
  'value',
  'onBlur',
  'onChange',
  'onFocus',
  'name',
  'checked',
  'disabled'
]

class CheckboxField extends React.Component {
  render() {
    const fieldProps = pick(this.props.fieldProps, CHECKBOX_FIELD_PROPS)

    return (
      <div>
        <label className={styles.label}>
          <input type='checkbox' {...fieldProps} />
          <span className={styles.title}>{this.props.title}</span>

          {this.props.hint && <div className={styles.hint}>{this.props.hint}</div>}
        </label>
      </div>
    )
  }
}

export default CheckboxField
