import React from 'react'

export default class RoutingContainer extends React.Component {
  render() {
    return (
      <div className='section-container'>
        {this.props.children}
      </div>
    )
  }
}
