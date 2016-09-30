import React from 'react'
import styles from './Main.scss'
import { Flash } from '../../../../shared'
import { Link } from 'react-router'
import { connect } from 'react-redux'
import { actions } from '../../../'

class Main extends React.Component {
  render() {
    let logo = require('../../../../../assets/images/logo-white.png')

    const mainStyles = [styles.main]
    if (navigator.userAgent.includes('ChainCore.app')) {
      mainStyles.push(styles.mainEmbedded)
    }

    return (
      <div className={mainStyles.join(' ')}>
        <div className={styles.sidebar}>
          <div className={styles.sidebarContent}>
            <div className={styles.logo}>
              <Link to={'/'}>
                <img src={logo} className={styles.brand_image} />
              </Link>
            </div>

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
                <Link to='/core' activeClassName={styles.active}>
                  <span className={`glyphicon glyphicon-hdd ${styles.glyphicon}`} />
                  Core
                </Link>
              </li>
            </ul>
          </div>
        </div>
        <div className={styles.content}>
          <Flash messages={this.props.flashMessages}
            markFlashDisplayed={this.props.markFlashDisplayed}
            dismissFlash={this.props.dismissFlash}
          />

          {this.props.children}
        </div>
      </div>
    )
  }
}

export default connect(
  (state) => ({
    flashMessages: state.app.flashMessages,
  }),
  (dispatch) => ({
    markFlashDisplayed: (key) => dispatch(actions.displayedFlash(key)),
    dismissFlash: (key) => dispatch(actions.dismissFlash(key)),
  })
)(Main)
