import * as React from 'react'
import { Link } from 'react-router-dom'
import ReactTooltip from 'react-tooltip'

import { prefixRoute } from '../../util'
import Reset from './reset'
import Seed from './seed'

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
        <ReactTooltip id="seedButtonTooltip" place="bottom" type="error" effect="solid"/>
        <ul className="nav navbar-nav navbar-right">
          <li><Link to={prefixRoute('/')}>Create</Link></li>
          <li><Link to={prefixRoute('/spend')}>Spend</Link></li>
          <li><Seed /></li>
          <li><Reset /></li>
        </ul>
      </div>
    </nav>
  )
}
