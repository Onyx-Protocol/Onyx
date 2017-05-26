// external imports
import * as React from 'react'
import { connect } from 'react-redux'

// internal imports
import { getSpendContract, getSpendContractId, getSelectedClauseIndex } from '../selectors'
import { setClauseIndex } from '../actions'
import { Clause } from '../../templates/types'

const ClauseSelect = (props: { contractId: string, clauses: Clause[],
                               setClauseIndex: (number)=>undefined, spendIndex: number }) => {
  return (
    <section>
      <h4>Clause</h4>
      <select className="form-control" value={props.spendIndex} onChange={(e) => props.setClauseIndex(e.target.value)}>
        {props.clauses.map((clause, i) => <option key={clause.name} value={i}>{clause.name}</option>)}
      </select>
    </section>
  )
}

export default connect(
  (state) => ({
    spendIndex: getSelectedClauseIndex(state),
    clauses: getSpendContract(state).template.clauses,
    contractId: getSpendContractId(state)
  }),
  { setClauseIndex }
)(ClauseSelect)
