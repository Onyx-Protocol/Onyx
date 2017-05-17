import * as React from 'react'
import NavBar from './navbar'

type Props = { children?: any }

export default function Root(props: Props) {
  return (
    <div>
      <NavBar />
      <div className="container fixedcontainer">
        {props.children}
      </div>
      <footer>
        <div className="container fixedcontainer">
          &copy; 2017 Chain Inc
        </div>
      </footer>
    </div>
  )
}
