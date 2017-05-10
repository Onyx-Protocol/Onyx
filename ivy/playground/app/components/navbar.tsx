import * as React from 'react'
import { Link } from 'react-router-dom'
import Reset from './reset'
import { prefixRoute } from '../../util'

const logo = require('../../static/images/logo.png')

export default function Navbar() {
  return (
    <nav className="navbar navbar-inverse navbar-static-top">
      <div className="container fixedcontainer">
        <div className="navbar-header">
          <a className="navbar-brand" href={prefixRoute('/')}>
            <img src={logo} />
          </a>
        </div>
        <ul className="nav navbar-nav navbar-right">
          <li><Link to={prefixRoute('/')}>Draft</Link></li>
          <li><Link to={prefixRoute('/spend')}>Spend</Link></li>
          <li><Reset /></li>
        </ul>
      </div>
    </nav>
  )
}
