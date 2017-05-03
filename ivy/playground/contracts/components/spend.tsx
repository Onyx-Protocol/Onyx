import * as React from 'react'
import { connect } from 'react-redux'
import Section from '../../app/components/section'
import { Item as Contract } from '../types'
import { Template, TemplateClause } from 'ivy-compiler'

import DocumentTitle from 'react-document-title'
// import SpendInputs from './spendinputs'
import SpendInputs from './argsDisplay'
import SpendFooter from './spendfooter'
import { DisplaySpendContract } from './display'
import Return from './return'
import ClauseSelect from './clauseselect'
import { getSpendContractId } from '../selectors'
import { ClauseParameters, getWidget } from './parameters'

export default function Spend(props: { contract: Contract }) {
  let contract = props.contract
  return (
    <DocumentTitle title="Spend">
      <Section name="Spend Contract" footer={<SpendFooter />}>
        <div className="form-wrapper">
        <section>
        <h4>Contract Template</h4>
        <DisplaySpendContract />
        </section>
        <ClauseSelect />
        <SpendInputs />
        <ClauseParameters />
        <Return />
        </div>
      </Section>
    </DocumentTitle>
  )
}