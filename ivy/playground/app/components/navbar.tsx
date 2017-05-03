import * as React from 'react'
import { Link } from 'react-router-dom'
import Reset from './reset'

export default function Navbar() {
  return (
    <nav className="navbar navbar-inverse navbar-static-top">
      <div className="container fixedcontainer">
        <div className="navbar-header">
          <a className="navbar-brand" href="/">
            <img src="/images/logo.png"/>
          </a>
        </div>
        <ul className="nav navbar-nav navbar-right">
          <li><Link to="/">Contract Templates</Link></li>
          <li><Link to="/create">Create Contract</Link></li>
          <li><Link to="/spend">Spend Contract</Link></li>
          <li><Reset /></li>
        </ul>
      </div>
    </nav>
  )
}
