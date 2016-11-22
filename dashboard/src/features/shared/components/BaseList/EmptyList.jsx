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
      emptyBlock = <div className={styles.objectHelp}>{this.props.emptyContent}</div>
    } else if (this.props.showFirstTimeFlow) {
      emptyBlock = <div>
        {this.props.showWelcome && <h2>Welcome to Chain Core!</h2>}
        <div className={styles.objectHelp}>
          {this.props.emptyContent}
          {this.props.newButton}
        </div>
      </div>
    } else if (!this.props.showFirstTimeFlow) {
      classNames.push(styles.noResults)
      emptyBlock = <div>
        <span className={`${styles.emptyLabel} ${styles.noResultsLabel}`}>No results for query:</span>
        <code className={styles.code}>{this.props.currentFilter.filter}</code>
        <div className={styles.objectHelp}>
          <p>To learn how to query the API, please refer to the documentation:</p>
          <ol>
            <li><a href="/docs/core/build-applications/queries" target="_blank">Queries</a></li>
            <li><a href="/docs/core/reference/api-objects" target="_blank">API Objects</a></li>
          </ol>
        </div>
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
