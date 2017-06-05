import * as React from 'react'

export default function Display(props: { source: string }) {
  return <pre>{props.source}</pre>
}
