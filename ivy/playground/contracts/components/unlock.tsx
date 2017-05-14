// external imports
import * as React from 'react'
import DocumentTitle from 'react-document-title'
import { connect } from 'react-redux'

// ivy imports
import Section from '../../app/components/section'
import { Contract } from '../types'
import { getSpendContractId } from '../selectors'

// internal imports
import SpendInputs from './argsDisplay'
import UnlockButton from './unlockButton'
import { DisplaySpendContract } from './display'
import UnlockDestination from './unlockDestination'
import ClauseSelect from './clauseselect'
import { ClauseValue, ClauseParameters, getWidget } from './parameters'
import { ContractValue } from './argsDisplay'

export default function Unlock(props: { contract: Contract }) {
  const contract = props.contract
  return (
    <DocumentTitle title="Unlock Value">
      <div>
        <Section name="Contract Summary">
          <div className="form-wrapper with-subsections">
            <section>
              <h4>Contract Template</h4>
              <DisplaySpendContract />
            </section>
            <ContractValue />
            <SpendInputs />
          </div>
        </Section>
        <Section name="Unlocking Details">
          <div className="form-wrapper with-subsections">
            <ClauseSelect />
            <ClauseValue />
            <ClauseParameters />
            <UnlockDestination />
          </div>
        </Section>
        <UnlockButton />
      </div>
    </DocumentTitle>
  )
}
