import React from 'react'
import styles from './EmptyList.scss'

class EmptyList extends React.Component {
  render() {
    let classNames = [styles.empty]
    let emptyImage
    try {
      emptyImage = require(`assets/images/empty/${this.props.type}.svg`)
    } catch (err) { /* do nothing */ }

    let emptyBlock
    if (!this.props.loadedOnce) {
      emptyBlock = <span>LOADING…</span>
    } else if (this.props.showFirstTimeFlow && this.props.skipCreate) {
      emptyBlock = <div>{this.props.emptyContent}</div>
    } else if (this.props.showFirstTimeFlow) {
      emptyBlock = <div>
        {this.props.emptyContent}
        {this.props.newButton}
      </div>
    } else if (!this.props.showFirstTimeFlow) {
      classNames.push(styles.noResults)
      emptyBlock = <div className={styles.emptyContainer}>
        <span className={`${styles.emptyLabel} ${styles.noResultsLabel}`}>No results for query:</span>
        <code className={styles.code}>{this.props.currentFilter.filter}</code>
      </div>
    }

    return (
      <div className={`flex-container ${classNames.join(' ')}`}>
        {emptyImage && <img className={styles.image} src={emptyImage} />}
        {emptyBlock}
      </div>
    )
  }
}

EmptyList.propTypes = {
  type: React.PropTypes.string,
  objectName: React.PropTypes.string,
  newButton: React.PropTypes.object,
  noRecords: React.PropTypes.bool,
  skipCreate: React.PropTypes.bool,
  loadedOnce: React.PropTypes.bool,
  currentFilter: React.PropTypes.object,
}

export default EmptyList
