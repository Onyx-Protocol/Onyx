import React from 'react'
import pick from 'lodash/pick'
import styles from './CheckboxField.scss'

const CHECKBOX_FIELD_PROPS = [
  'value',
  'onBlur',
  'onChange',
  'onFocus',
  'name'
]

class CheckboxField extends React.Component {
  render() {
    const fieldProps = pick(this.props.fieldProps, CHECKBOX_FIELD_PROPS)

    return (
      <div>
        <label>
          <input type='checkbox' {...fieldProps} />
          {this.props.title}
        </label>

        {this.props.hint && <span className='help-block'>{this.props.hint}</span>}
      </div>
    )
  }
}

export default CheckboxField
