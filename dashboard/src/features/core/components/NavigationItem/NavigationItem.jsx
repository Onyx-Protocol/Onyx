import { connect } from 'react-redux'
import { getUpcomingReset, getNetworkMismatch } from 'features/configuration/reducers'
import React from 'react'
import styles from './NavigationItem.scss'

class NavigationItem extends React.Component {
  render() {
    return(
      <span className={styles.main}>
        <span className={`glyphicon glyphicon-hdd ${this.props.externalStyles}`} />
        Core

        {this.props.showWarning && <span className={`glyphicon glyphicon-warning-sign ${styles.warning}`} />}
      </span>
    )
  }
}

export default connect(
  (state) => {
    const networkMismatch = getNetworkMismatch(state)
    const resetUpcoming = getUpcomingReset(state)

    return {
      showWarning: state.core.onTestNet && (networkMismatch || resetUpcoming)
    }
  }
)(NavigationItem)
