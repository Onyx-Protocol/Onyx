import * as React from 'react'
import NavBar from './navbar'
import ReactTooltip from 'react-tooltip'

type Props = { children?: any }

export default function Root(props: Props) {
  return (
    <div>
      <ReactTooltip place="bottom" type="error" effect="solid"/>
      <NavBar />
      <div className="container fixedcontainer">
        {props.children}
      </div>
    </div>
  )
}
