import React from 'react'

class Flash extends React.Component {
  render() {
    if (!this.props.message) {
      return null
    }

    return (
      <div className={`alert alert-${this.props.type}`}>
        {this.props.message}

        <button type="button" className="close" onClick={this.props.dismissFlash}>
          <span>&times;</span>
        </button>
      </div>
    )
  }
}

export default Flash
