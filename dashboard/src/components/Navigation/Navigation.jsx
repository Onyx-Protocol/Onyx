import React from 'react'
import { Link } from 'react-router'
import styles from "./Navigation.scss"

class Navigation extends React.Component {
  constructor(props) {
    super(props)
    this.state = { dropdownClass: "" }

    this.toggleDropdown = this.toggleDropdown.bind(this)
    this.closeDropdown = this.closeDropdown.bind(this)
  }

  toggleDropdown(event) {
    event.stopPropagation()

    let existing = this.state.dropdownClass
    this.setState({ dropdownClass: (existing == "" ? "open" : "") })
  }

  closeDropdown() {
    this.setState({ dropdownClass: "" })
  }

  render() {
    let logo = require('../../images/logo-white.png')
    return (
      <div onClick={this.closeDropdown}>
        <div className="navbar navbar-default navbar-fixed-top">
          <div className="container">
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

                <li className={`dropdown ${this.state.dropdownClass}`}>
                  <a href="#" onClick={this.toggleDropdown}>
                    <span className={`glyphicon glyphicon-cog ${styles.glyphicon}`} />
                      Configuration <span className="caret">
                    </span>
                  </a>

                  <ul className={`dropdown-menu ${styles.dropdown}`}>
                    <li>
                      <Link to={`/indexes`} activeClassName={styles.active}>
                        <span className={`glyphicon glyphicon-eye-open ${styles.glyphicon}`} />
                        Indexes
                      </Link>
                    </li>
                    <li>
                      <Link to={`/mockhsms`} activeClassName={styles.active}>
                        <span className={`glyphicon glyphicon-lock ${styles.glyphicon}`} />
                        Mock HSM
                      </Link>
                    </li>
                    <li>
                      <Link to={`/core-settings`} activeClassName={styles.active}>
                        <span className={`glyphicon glyphicon-hdd ${styles.glyphicon}`} />
                        Core Settings
                      </Link>
                    </li>
                  </ul>
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
            © Chain
          </div>
        </footer>
      </div>
    )
  }
}

export default Navigation
