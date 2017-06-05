// external imports
import * as React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import ReactTooltip from 'react-tooltip'

// ivy imports
import { prefixRoute } from '../../core'
import { isFirstTime } from '../../contracts/selectors'

// internal imports
import Reset from './reset'
import Seed from './seed'

const logo = require('../../../static/images/logo.png')
const symbol = require('../../../static/images/chain-symbol.svg')

import { closeModal } from '../../contracts/actions'

const mapStateToProps = (state) => {
  const location = state.routing.location
  if (!location) {
    return { path: 'lock' }
  }

  const pathname = location.pathname.split("/")
  if (pathname[1] === "ivy") {
    pathname.shift()
  }
  return { path: pathname[1], firstTime: isFirstTime(state) }
}

const Navbar = (props: { path: string, firstTime: boolean, closeModal: ()=>undefined }) => {
  return (
    <nav className="navbar navbar-inverse navbar-static-top navbar-fixed-top">
      <div className="container fixedcontainer">
        <div className="navbar-header">
          <a className="navbar-brand" href={prefixRoute('/')}>
            <img src={logo} />
          </a>
        </div>
        <ul className="nav navbar-nav navbar-right">
          <li className={props.path === 'unlock' ? '' : 'active'} ><Link to={prefixRoute('/')}>Lock Value</Link></li>
          <li className={props.path === 'unlock' ? 'active' : ''} ><Link to={prefixRoute('/unlock')}>Unlock Value</Link></li>
          <li className="divider-vertical"></li>
          <li><a href="https://chain.com/docs/1.2/ivy-playground/docs" target="_blank">Docs</a></li>
          <li><a href="https://chain.com/docs/1.2/ivy-playground/tutorial" target="_blank">Tutorial</a></li>
          <li><a href="../dashboard" target="_blank">Dashboard</a></li>
          <li className="dropdown">
            <a href="#" className="dropdown-toggle" data-toggle="dropdown" role="button" aria-haspopup="true" aria-expanded="false">Setup <span className="caret"></span></a>
            <ul className="dropdown-menu">
              {/* Reset and Seed return <li> elements */}
              <Reset />
            </ul>
          </li>
        </ul>
        <div className="welcome" hidden={!props.firstTime}>
        <div className="welcome-content">
          <img src={symbol}/>
          <h1>Welcome to Ivy Playground!</h1>
          <p>The <a href="https:/chain.com/docs/1.2/ivy-playground/tutorial" target="_blank">tutorial</a> is a great place to start. It will teach how to write a contract in Ivy, and then lock and unlock assets with it.</p>
          <p>The Ivy Playground interacts with Chain Core and the MockHSM to build and sign transactions. We created a few accounts and assets to help you get started. Enjoy!</p>
          <button className="btn btn-primary btn-xl" onClick={props.closeModal}>Let's Go!</button>
        </div>
        <div className="welcome-screen-block"></div>
         </div>
      </div>
    </nav>
  )
}

export default connect(
  mapStateToProps,
  { closeModal }
)(Navbar)
