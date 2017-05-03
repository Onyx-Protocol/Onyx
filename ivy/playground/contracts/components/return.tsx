import * as React from 'react'
import { connect } from 'react-redux'
import { TemplateClause } from 'ivy-compiler'

import { getWidget } from '../../contracts/components/parameters'
import { getSpendTemplateClause } from '../selectors'


const Return = (props: { clause: TemplateClause }) => {
  if (props.clause.returnStatement === undefined) {
    return <div></div>
  } else {
    return (
      <section>
        <h4>Return Destination</h4>
        {getWidget("transactionDetails.accountAliasInput")}
      </section>
    )
  }
}

export default connect(
  (state) => ({ clause: getSpendTemplateClause(state) })
)(Return)