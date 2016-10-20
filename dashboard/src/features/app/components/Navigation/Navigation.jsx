import React from 'react'
import { Link } from 'react-router'
import styles from './Navigation.scss'

class Navigation extends React.Component {
  render() {
    return (
      <div className={styles.main}>
        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>blockchain data</li>
          <li>
            <Link to='/transactions' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-transfer ${styles.glyphicon}`} />
              Transactions
            </Link>
          </li>
          <li>
            <Link to='/assets' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-file ${styles.glyphicon}`} />
              Assets
            </Link>
          </li>
          <li>
            <Link to='/accounts' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-user ${styles.glyphicon}`} />
              Accounts
            </Link>
          </li>
          <li>
            <Link to='/balances' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-stats ${styles.glyphicon}`} />
              Balances
            </Link>
          </li>
          <li>
            <Link to='/unspents' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-th-list ${styles.glyphicon}`} />
              Unspent Outputs
            </Link>
          </li>
        </ul>

        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>configuration</li>
          <li>
            <Link to='/mockhsms' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-lock ${styles.glyphicon}`} />
              Mock HSM
            </Link>
          </li>
          <li>
            <Link to='/transaction-feeds' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-th-list ${styles.glyphicon}`} />
              Feeds
            </Link>
          </li>
        </ul>
        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>developers</li>
          <li>
            <a href='/docs' target='_blank'>
              <span className={`glyphicon glyphicon-book ${styles.glyphicon}`} />
              Documentation
            </a>
          </li>
          <li>
            <a href='https://chain.com/support' target='_blank'>
              <span className={`glyphicon glyphicon-earphone ${styles.glyphicon}`} />
              Support
            </a>
          </li>
        </ul>
      </div>
    )
  }
}

export default Navigation
