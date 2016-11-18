import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router'
import styles from './Navigation.scss'
import Sync from '../Sync/Sync'
import { navIcon } from '../../utils'

class Navigation extends React.Component {
  render() {
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
              Unspent outputs
            </Link>
          </li>
        </ul>

        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>services</li>
          <li>
            <Link to='/mockhsms' activeClassName={styles.active}>
              {navIcon('mockhsm', styles)}
              MockHSM
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

        {this.props.showSync && <Sync />}
      </div>
    )
  }
}

export default connect(
  state => ({
    routing: state.routing, // required for <Link>s to update active state on navigation
    showSync: state.core.configured && !state.core.generator,
  })
)(Navigation)
