import React from 'react'
import { Navbar } from '../Common'
import styles from './Config.scss'

class Config extends React.Component {
  render() {
    return (
      <div>
        <Navbar customStyles='navbar-static-top'>
          <div className={`navbar-header ${styles.header}`}>
            <div className={`navbar-brand ${styles.title}`}>
              Configure Chain Core
            </div>
          </div>
        </Navbar>

        <div className='container'>
          {this.props.children}
        </div>

      </div>
    )
  }
}

export default Config
