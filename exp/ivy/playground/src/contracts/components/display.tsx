import * as React from 'react'
import { connect } from 'react-redux'
import { getSpendContract } from '../selectors'

export const Display = (props: { source: string }) => {
  return <pre className="codeblock">{props.source}</pre>
}

export const DisplaySpendContract = connect(
  (state) => {
    const contract = getSpendContract(state)
    if (contract) {
      return { source: contract.template.source }
    }
    return { source: '' }
  }
)(Display)
