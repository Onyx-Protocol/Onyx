import React from 'react'
import styles from './TextField.scss'
import pick from 'lodash.pick'
import { FieldLabel } from 'features/shared/components'

const TEXT_FIELD_PROPS = [
  'value',
  'onBlur',
  'onChange',
  'onFocus',
]

class TextField extends React.Component {
  constructor(props) {
    super(props)
    this.state = {type: 'text'}
  }

  render() {
    const fieldProps = pick(this.props.fieldProps, TEXT_FIELD_PROPS)

    const inputClasses = ['form-control']
    const error = this.props.fieldProps.error
    if (error) {
      inputClasses.push(styles.errorInput)
    }

    return(
      <div className='form-group'>
        {this.props.title && <FieldLabel className={styles.title}>{this.props.title}</FieldLabel>}
        <input className='form-control'
          type={this.state.type}
          placeholder={this.props.placeholder}
          autoFocus={!!this.props.autoFocus}
          {...fieldProps} />

        {this.props.hint && <span className='help-block'>{this.props.hint}</span>}
        {error && <span className={styles.errorText}>{error}</span>}
      </div>
    )
  }
}

export default TextField
