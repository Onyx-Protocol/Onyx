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
