import * as React from 'react'
import { connect } from 'react-redux'

import { TemplateClause } from 'ivy-compiler'

import { getSpendContract, getSpendContractId, getSelectedClauseIndex } from '../selectors'
import { setClauseIndex } from '../actions'

const ClauseSelect = (props: { contractId: string, clauses: TemplateClause[], 
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
    clauses: getSpendContract(state).template.clauseInfo,
    contractId: getSpendContractId(state)
  }),
  { setClauseIndex }
)(ClauseSelect)
