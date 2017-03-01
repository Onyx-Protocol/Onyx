import React from 'react'
import styles from './EmptyContent.scss'

class EmptyContent extends React.Component {
  render() {

    return (
      <div className={styles.emptyContainer}>
        {this.props.children && <div className={styles.emptyContent}>
          {this.props.children}
        </div>}
      </div>
    )
  }
}

EmptyContent.propTypes = {
  title: React.PropTypes.string
}

export default EmptyContent
