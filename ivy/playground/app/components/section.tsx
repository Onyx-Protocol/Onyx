import * as React from 'react'

type Props = {
  name: string,
  footer?: JSX.Element,
  children?: any
}

export default function Section(props: Props) {
  return (
    <div className="panel panel-default">
      <div className="panel-heading">
        <h1 className="panel-title">{props.name}</h1>
      </div>
      <div className="panel-body">
        { props.children }
      </div>
      { props.footer ? <div className="panel-footer">{props.footer}</div> : <div /> }
    </div>
  )
}
