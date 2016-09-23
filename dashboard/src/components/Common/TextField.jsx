import React from 'react'

class TextField extends React.Component {
  constructor(props) {
    super(props)
    this.state = {type: 'text'}
  }

  render() {
    return(
      <div className='form-group'>
        {this.props.title && <label>{this.props.title}</label>}
        <input className='form-control'
          type={this.state.type}
          placeholder={this.props.placeholder}
          {...this.props.fieldProps} />

        {this.props.hint && <span className='help-block'>{this.props.hint}</span>}
      </div>
    )
  }
}

export default TextField
