import React from 'react'
import styles from './FormSection.scss'

class FormSection extends React.Component {
  render() {
    return (
      <div className={`${styles.main} ${this.props.className || ''}`}>
        <div className={styles.title}>{this.props.title}</div>

        <div className={styles.content}>
          {this.props.children}
        </div>
      </div>
    )
  }
}

export default FormSection
