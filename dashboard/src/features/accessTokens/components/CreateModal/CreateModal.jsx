import React from 'react'
import { copyToClipboard } from 'utility/clipboard'
import styles from './CreateModal.scss'

export default class CreateModal extends React.Component {
  copyClick() {
    copyToClipboard(this.props.token)
  }

  render() {
    return <div>
      <h4>Created new access token</h4>
      <p>Please store this token carefully. This is the last time it will be displayed.</p>
      <div className={styles.tokenContainer}>
        <pre className={styles.pre}>{this.props.token}</pre>
        <div className={styles.copyButton}>
          <button className='btn btn-default btn-sm' onClick={this.copyClick.bind(this)}>Copy to clipboard</button>
        </div>
      </div>
    </div>
  }
}
