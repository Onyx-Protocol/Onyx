import React from 'react'
import styles from './TextField.scss'

class TextField extends React.Component {
  constructor(props) {
    super(props)
    this.state = {type: 'text'}
  }

  render() {
    const inputClasses = ['form-control']
    const error = this.props.fieldProps.error
    if (error) {
      inputClasses.push(styles.errorInput)
    }

    return(
      <div>
        <div className='form-group'>
          {this.props.title && <label>{this.props.title}</label>}
          <input className='form-control'
            type={this.state.type}
            placeholder={this.props.placeholder}
            autoFocus={!!this.props.autoFocus}
            {...this.props.fieldProps} />

          {this.props.hint && <span className='help-block'>{this.props.hint}</span>}
        </div>

        {error && <span className={styles.errorText}>{error}</span>}
      </div>
    )
  }
}

export default TextField
