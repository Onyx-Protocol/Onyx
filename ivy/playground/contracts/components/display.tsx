import * as React from 'react'
import { connect } from 'react-redux'
import { getSpendContract } from '../selectors'

export const Display = (props: { source: string }) => {
  return <pre className="codeblock">{props.source}</pre>
}

export const DisplaySpendContract = connect(
  (state) => ({ source: getSpendContract(state).template.source })
)(Display)
