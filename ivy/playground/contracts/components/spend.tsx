import * as React from 'react'
import { connect } from 'react-redux'
import Section from '../../app/components/section'
import { Contract } from '../types'
import { Template, TemplateClause } from 'ivy-compiler'

import DocumentTitle from 'react-document-title'
import SpendInputs from './argsDisplay'
import SpendButton from './spendbutton'
import { DisplaySpendContract } from './display'
import Return from './return'
import ClauseSelect from './clauseselect'
import { getSpendContractId } from '../selectors'
import { ClauseParameters, getWidget } from './parameters'

export default function Spend(props: { contract: Contract }) {
  let contract = props.contract
  return (
    <DocumentTitle title="Spend Contract">
      <div>
        <Section name="Contract Summary">
          <div className="form-wrapper">
            <section>
              <h4>Contract Template</h4>
              <DisplaySpendContract />
            </section>
            <SpendInputs />
          </div>
        </Section>
        <Section name="Spending Details">
          <div className="form-wrapper">
            <ClauseSelect />
            <ClauseParameters />
            <Return />
          </div>
        </Section>
        <SpendButton />
      </div>
    </DocumentTitle>
  )
}
