import React from 'react'
import styles from './PageContent.scss'

class PageContent extends React.Component {
  render() {
    return (
      <div className={styles.main}>
        {this.props.children}
      </div>
    )
  }
}

export default PageContent
