import React from 'react'

class Config extends React.Component {
  render() {
    return (
      <div>
        <div className='container'>
          {this.props.children}
        </div>
      </div>
    )
  }
}

export default Config
