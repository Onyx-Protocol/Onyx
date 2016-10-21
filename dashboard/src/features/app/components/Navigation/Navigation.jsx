import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router'
import styles from './Navigation.scss'
import { humanizeDuration } from 'utility/time'

export const navIcon = (name, styles) => {
  let active = false
  const icon = require(`assets/images/navigation/${name}.png`)

  try {
    active = require(`assets/images/navigation/${name}-active.png`)
  } catch (err) { /* do nothing */ }
  return (
    <span className={styles.iconWrapper}>
      <img className={styles.icon} src={icon}/>
      {active && <img className={styles.activeIcon} src={active}/>}
    </span>
  )
}

class Navigation extends React.Component {
  render() {
    const {
      replicationLag,
      showSync,
      snapshot,
      syncEstimates,
    } = this.props

    let syncContent

    if (showSync) {
      if (snapshot && snapshot.in_progress) { // Currently downloading the snapshot.
        const downloaded = (snapshot.downloaded / snapshot.size) * 100
        syncContent = <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>snapshot sync</li>
          <li>{snapshot.height} blocks</li>
          <li>{Math.round(downloaded)}% downloaded</li>
          {!!syncEstimates.snapshot && <li>Time remaining: {humanizeDuration(syncEstimates.snapshot)}</li>}
        </ul>
      } else if (replicationLag !== null && replicationLag < 3) { // synced up, or close to it
        syncContent = <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>generator sync</li>
          <li>Local core fully synced.</li>
        </ul>
      } else { // Using RPC sync
        // TODO(jeffomatic): Show a warning if the snapshot did not succeed.
        syncContent = <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>generator sync</li>
          <li>Blocks behind: {replicationLag === null ? '-' : replicationLag}</li>
          {!!syncEstimates.replicaLag && <li>Time remaining: {humanizeDuration(syncEstimates.replicaLag)}</li>}
        </ul>
      }
    }

    return (
      <div className={styles.main}>
        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>core data</li>
          <li>
            <Link to='/transactions' activeClassName={styles.active}>
              {navIcon('transaction', styles)}
              Transactions
            </Link>
          </li>
          <li>
            <Link to='/accounts' activeClassName={styles.active}>
              {navIcon('account', styles)}
              Accounts
            </Link>
          </li>
          <li>
            <Link to='/assets' activeClassName={styles.active}>
              {navIcon('asset', styles)}
              Assets
            </Link>
          </li>
          <li>
            <Link to='/balances' activeClassName={styles.active}>
              {navIcon('balance', styles)}
              Balances
            </Link>
          </li>
          <li>
            <Link to='/unspents' activeClassName={styles.active}>
              {navIcon('unspent', styles)}
              Unspent Outputs
            </Link>
          </li>
        </ul>

        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>services</li>
          <li>
            <Link to='/mockhsms' activeClassName={styles.active}>
              {navIcon('mockhsm', styles)}
              Mock HSM
            </Link>
          </li>
          <li>
            <Link to='/transaction-feeds' activeClassName={styles.active}>
              {navIcon('feed', styles)}
              Feeds
            </Link>
          </li>
        </ul>
        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>developers</li>
          <li>
            <a href='/docs' target='_blank'>
              {navIcon('docs', styles)}
              Documentation
            </a>
          </li>
          <li>
            <a href='https://chain.com/support' target='_blank'>
              {navIcon('help', styles)}
              Support
            </a>
          </li>
        </ul>

        {syncContent}
      </div>
    )
  }
}

export default connect(
  (state) => ({
    routing: state.routing, // required for <Link>s to update active state on navigation
    replicationLag: state.core.replicationLag,
    showSync: state.core.configured && !state.core.generator,
    snapshot: state.core.snapshot,
    syncEstimates: state.core.syncEstimates,
  })
)(Navigation)
