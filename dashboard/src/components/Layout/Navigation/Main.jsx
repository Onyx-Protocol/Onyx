import React from 'react'
import { Navbar } from '../../Common'
import { Link } from 'react-router'
import styles from './Main.scss'

class Navigation extends React.Component {
  render() {
    let logo = require('../../../assets/images/logo-white.png')

    return (
      <Navbar customStyles={`navbar-fixed-top ${styles.navbar_fixed}`}>
        <div className="navbar-header">
          <Link to={'/'} className="navbar-brand">
            <img src={logo} className={styles.brand_image} />
          </Link>
        </div>

        <div className="collapse navbar-collapse">
          <ul className="nav navbar-nav navbar-right">
            <li>
              <Link to={`/transactions`} activeClassName={styles.active}>
                <span className={`glyphicon glyphicon-transfer ${styles.glyphicon}`} />
                 Transactions
              </Link>
            </li>
            <li>
              <Link to={`/assets`} activeClassName={styles.active}>
                <span className={`glyphicon glyphicon-file ${styles.glyphicon}`} />
                Assets
              </Link>
            </li>
            <li>
              <Link to={`/accounts`} activeClassName={styles.active}>
                <span className={`glyphicon glyphicon-user ${styles.glyphicon}`} />
                Accounts
              </Link>
            </li>
            <li>
              <Link to={`/balances`} activeClassName={styles.active}>
                <span className={`glyphicon glyphicon-stats ${styles.glyphicon}`} />
                Balances
              </Link>
            </li>
            <li>
              <Link to={`/unspents`} activeClassName={styles.active}>
                <span className={`glyphicon glyphicon-th-list ${styles.glyphicon}`} />
                Unspent Outputs
              </Link>
            </li>
            <li className={styles.divider}>|</li>

            <li className={`dropdown ${this.props.dropdownState}`}>
              <a href="#" onClick={this.props.toggleDropdown}>
                <span className={`glyphicon glyphicon-cog ${styles.glyphicon}`} />
                  Configuration <span className="caret">
                </span>
              </a>

              <ul className={`dropdown-menu ${styles.dropdown}`}>
                <li>
                  <Link to={`/mockhsms`} activeClassName={styles.active}>
                    <span className={`glyphicon glyphicon-lock ${styles.glyphicon}`} />
                    Mock HSM
                  </Link>
                </li>
                <li>
                  <Link to={`/core`} activeClassName={styles.active}>
                    <span className={`glyphicon glyphicon-hdd ${styles.glyphicon}`} />
                    Core
                  </Link>
                </li>
              </ul>
            </li>
          </ul>
        </div>
      </Navbar>
    )
  }
}

export default Navigation
