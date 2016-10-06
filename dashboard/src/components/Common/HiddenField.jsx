import React from 'react'

class HiddenField extends React.Component {
  render() {
    return(
      <input className='form-control'
        type='hidden'
        onChange={this.props.fieldProps.onChange}
        value={this.props.fieldProps.value} />
    )
  }

}

export default HiddenField
