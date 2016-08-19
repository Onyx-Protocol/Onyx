import React from 'react'

export default class SectionContainer extends React.Component {
  render() {
    return (
      <div className="section-container">
        {this.props.children}
      </div>
    )
  }
}
