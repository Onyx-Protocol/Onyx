import React from 'react'
import styles from './FieldLabel.scss'

class FieldLabel extends React.Component {
  render() {
    return (<label className={styles.main}>{this.props.children}</label>)
  }
}

export default FieldLabel
