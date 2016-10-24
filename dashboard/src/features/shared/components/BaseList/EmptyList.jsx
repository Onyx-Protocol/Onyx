import React from 'react'
import styles from './EmptyList.scss'

class EmptyList extends React.Component {
  render() {
    let emptyImage
    try {
      emptyImage = require(`assets/images/empty/${this.props.type}.svg`)
    } catch (err) { /* do nothing */ }

    let emptyBlock
    if (!this.props.loadedOnce) {
      emptyBlock = <span>LOADING…</span>
    } else if (this.props.showFirstTimeFlow && this.props.skipCreate) {
      emptyBlock = <div>
        <span className={styles.emptyLabel}>
          There are no {this.props.objectName.toLowerCase()}s on the blockchain
        </span>
      </div>
    } else if (this.props.showFirstTimeFlow) {
      emptyBlock = <div>
        <span className={styles.emptyLabel}>Create your first {this.props.objectName}</span>
        {this.props.newButton}
      </div>
    } else if (!this.props.showFirstTimeFlow) {
      emptyBlock = <div>
        <span className={styles.emptyLabel}>No results for query:</span>
        <code className={styles.code}>{this.props.currentFilter.filter}</code>
      </div>
    }

    return (
      <div className={`flex-container ${styles.empty}`}>
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
