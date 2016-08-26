import React from 'react'

class TextField extends React.Component {
  constructor(props) {
    super(props)
    this.state = {type: "text"}
  }

  render() {
    return(
      <div className='form-group'>
        <label>{this.props.title}</label>
        <input className='form-control'
          type={this.state.type}
          {...this.props.fieldProps} />
      </div>
    )
  }
}

export default TextField
