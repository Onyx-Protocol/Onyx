import React from 'react'
import styles from './SubmitIndicator.scss'

export default class SubmitIndicator extends React.Component {
  render() {
    const text = this.props.text || 'Submitting...'
    return <div className={styles.activeSubmit}>{text}</div>
  }
}
