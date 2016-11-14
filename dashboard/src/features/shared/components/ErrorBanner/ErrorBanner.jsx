import React from 'react'
import styles from './ErrorBanner.scss'

class ErrorBanner extends React.Component {
  render() {
    const error = this.props.error || ''
    const message = error.chainMessage || error.message || error

    return (
      <div className={styles.main}>
        {this.props.title && <strong>{this.props.title}<br/></strong>}

        {message &&
          <div className={(error.code || error.requestId) ? styles.message : ''}>
            {message}{error.detail ? `: ${error.detail}` : ''}
          </div>}

        {error.code &&
          <div className={styles.extra}>Error Code: <strong>{error.code}</strong></div>}

        {error.requestId &&
          <div className={styles.extra}>Request ID: <strong>{error.requestId}</strong></div>}
      </div>
    )
  }
}

export default ErrorBanner
