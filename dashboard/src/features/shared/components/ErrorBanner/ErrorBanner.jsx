import React from 'react'
import styles from './ErrorBanner.scss'

class ErrorBanner extends React.Component {
  render() {
    const error = this.props.error

    return (
      <div className={styles.main}>
        {this.props.title && <strong>{this.props.title}<br/></strong>}

        <span>{error.chainMessage}{error.detail ? `: ${error.detail}` : ''}</span><br/>

        {error.code &&
          <span>Error Code: <strong>{error.code}</strong><br/></span>}

        {error.requestId &&
          <span>Request ID: <strong>{error.requestId}</strong><br/></span>}
      </div>
    )
  }
}

export default ErrorBanner
