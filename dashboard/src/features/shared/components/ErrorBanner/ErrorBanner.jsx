import React from 'react'
import styles from './ErrorBanner.scss'

class ErrorBanner extends React.Component {
  render() {
    return (
      <div className={styles.main}>
        {this.props.title && <strong>{this.props.title}<br/></strong>}
        {this.props.message}
      </div>
    )
  }
}

export default ErrorBanner
