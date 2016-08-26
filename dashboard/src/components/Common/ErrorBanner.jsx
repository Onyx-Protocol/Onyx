import React from 'react'

class ErrorBanner extends React.Component {
  render() {
    return (
      <div className="alert alert-danger">
        {this.props.title && <strong>{this.props.title}<br/></strong>}
        {this.props.message}
      </div>
    )
  }
}

export default ErrorBanner
