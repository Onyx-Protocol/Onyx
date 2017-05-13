import * as React from 'react'
import { Link } from 'react-router-dom'
import ReactTooltip from 'react-tooltip'

import { prefixRoute } from '../../util'
import Reset from './reset'
import Seed from './seed'

const logo = require('../../static/images/logo.png')

export default function Navbar() {
  return (
    <nav className="navbar navbar-inverse navbar-static-top navbar-fixed-top">
      <div className="container fixedcontainer">
        <div className="navbar-header">
          <a className="navbar-brand" href={prefixRoute('/')}>
            <img src={logo} />
          </a>
        </div>
        <ReactTooltip id="seedButtonTooltip" place="bottom" type="error" effect="solid"/>
        <ul className="nav navbar-nav navbar-right">
          <li><Link to={prefixRoute('/')}>Lock Value</Link></li>
          <li><Link to={prefixRoute('/unlock')}>Unlock Value</Link></li>
          <li className="divider-vertical"></li>
          <li><a href="https://chain.com/docs" target="_blank">Docs</a></li>
          <li className="dropdown">
            <a href="#" className="dropdown-toggle" data-toggle="dropdown" role="button" aria-haspopup="true" aria-expanded="false">Setup <span className="caret"></span></a>
            <ul className="dropdown-menu">
              {/* Reset and Seed return <li> elements */}
              <Reset />
              <Seed />
            </ul>
          </li>
        </ul>
      </div>
    </nav>
  )
}
