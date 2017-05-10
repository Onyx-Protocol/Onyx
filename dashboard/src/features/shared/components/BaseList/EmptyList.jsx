import React from 'react'
import styles from './EmptyList.scss'
import componentClassNames from 'utility/componentClassNames'
import { docsRoot } from 'utility/environment'

class EmptyList extends React.Component {
  render() {
    let emptyImage

    try {
      emptyImage = require(`images/empty/${this.props.type}.svg`)
    } catch (err) { /* do nothing */ }

    let emptyBlock
    if (!this.props.loadedOnce) {
      emptyBlock = <span>LOADING…</span>
    } else if (this.props.showFirstTimeFlow) {
      emptyBlock = <div>
        <span className={`${styles.emptyLabel} ${styles.noResultsLabel}`}>
          There are no {this.props.objectName}s
        </span>
        {this.props.firstTimeContent}
      </div>
    } else if (!this.props.showFirstTimeFlow) {
      emptyBlock = <div className={styles.emptyContainer}>
        <span className={`${styles.emptyLabel} ${styles.noResultsLabel}`}>No results for query:</span>
        <code className={styles.code}>{this.props.currentFilter.filter}</code>
        <div className={styles.emptyContent}>
          To learn how to query the API, please refer to the documentation:
          <ol>
            <li><a href={`${docsRoot}/core/build-applications/queries`} target='_blank'>Queries</a></li>
            <li><a href={`${docsRoot}/core/reference/api-objects`} target='_blank'>API Objects</a></li>
          </ol>
        </div>
      </div>
    }

    const classNames = [
      'flex-container',
      styles.empty,
      {[styles.noResults]: !this.props.showFirstTimeFlow}
    ]

    return (
      <div className={componentClassNames(this, ...classNames)}>
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
  showFirstTimeFlow: React.PropTypes.bool,
  firstTimeContent: React.PropTypes.object
}

export default EmptyList
