import React from 'react'

export default class TextareaField extends React.Component {
  render() {
    return(
      <div className='form-group'>
        <label>{this.props.title}</label>
        <textarea
          {...this.props.fieldProps}
          className='form-control'
          value={this.props.fieldProps.value || ''}
        />
      </div>
    )
  }
}
