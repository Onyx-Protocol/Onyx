import React from 'react'

class Panel extends React.Component {
  render() {
    return(
      <div className='panel panel-default'>
        {this.props.title && <div className='panel-heading'>{this.props.title}</div>}
        <div className='panel-body'>
          {this.props.children}
        </div>
      </div>
    )
  }
}

export default Panel
