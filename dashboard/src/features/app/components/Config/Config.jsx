import React from 'react'
import { Navbar } from 'components/Common'
import styles from './Config.scss'

class Config extends React.Component {
  render() {
    return (
      <div>
        <div className='container'>
          {this.props.children}
        </div>
      </div>
    )
  }
}

export default Config
