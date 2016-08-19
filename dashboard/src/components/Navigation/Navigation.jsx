import React from 'react'
import { Link } from 'react-router'
import styles from "./Navigation.scss"

class Navigation extends React.Component {
  render() {
    let logo = require('../../images/logo-white.png')
    return (
      <div>
        <div className="navbar navbar-default navbar-static-top">
          <div className="container">
            <div className="navbar-header">
              <Link to={'/'} className="navbar-brand">
                <img src={logo} className={styles.brand_image} />
              </Link>
            </div>

            <div className="collapse navbar-collapse">
              <ul className="nav navbar-nav">
                <li>
                  <Link to={`/transactions`}>
                    <span className={`glyphicon glyphicon-transfer ${styles.glyphicon}`} />
                     Transactions
                  </Link>
                </li>
                <li>
                  <Link to={`/unspents`}>
                    <span className={`glyphicon glyphicon-link ${styles.glyphicon}`} />
                    Unspent output
                  </Link>
                </li>
                <li>
                  <Link to={`/balances`}>
                    <span className={`glyphicon glyphicon-usd ${styles.glyphicon}`} />
                    Balances
                  </Link>
                </li>
                <li className={styles.divider}>|</li>
                <li>
                  <Link to={`/accounts`}>
                    <span className={`glyphicon glyphicon-user ${styles.glyphicon}`} />
                    Accounts
                  </Link>
                </li>
                <li>
                  <Link to={`/assets`}>
                    <span className={`glyphicon glyphicon-barcode ${styles.glyphicon}`} />
                    Assets
                  </Link>
                </li>
                <li>
                  <Link to={`/indexes`}>
                    <span className={`glyphicon glyphicon-eye-open ${styles.glyphicon}`} />
                    Indexes
                  </Link>
                </li>
                <li className={styles.divider}>|</li>
                <li>
                  <Link to={`/mockhsms`}>
                    <span className={`glyphicon glyphicon-lock ${styles.glyphicon}`} />
                    Mock HSM
                  </Link>
                </li>
              </ul>
            </div>
          </div>
        </div>

        <div className="container">
          {this.props.children}
        </div>

        <footer className={`${styles.footer}`}>
          <div className="container">
            ✨&nbsp;&nbsp;© Chain ✨
          </div>
        </footer>
      </div>
    )
  }
}

export default Navigation
