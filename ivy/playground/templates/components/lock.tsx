// external imports
import * as React from 'react'
import { connect } from 'react-redux'
import DocumentTitle from 'react-document-title'

import Section from '../../app/components/section'
import LockButton from './lockButton'

import { getSource, getContractParameters, getCompiled } from '../selectors'

import { ContractParameters, ContractValue } from '../../contracts/components/parameters'

import Editor from './editor'

import { Display } from '../../contracts/components/display'

const mapStateToProps = (state) => {
  const source = getSource(state)
  const contractParameters = getContractParameters(state)
  const instantiable = contractParameters !== undefined
  return { source, instantiable }
}

const Lock = ({ source, instantiable }) => {
  let instantiate
  if (instantiable) {
    instantiate = (
      <div>
        <Section name="Value to Lock">
          <div className="form-wrapper">
            <ContractValue />
          </div>
        </Section>
        <Section name="Contract Arguments">
          <div className="form-wrapper">
            <ContractParameters />
          </div>
        </Section>
        <LockButton />
      </div>
    )
  } else {
    instantiate = ( <div /> )
  }
  return (
    <DocumentTitle title='Lock Value'>
      <div>
        <Editor />
        {instantiate}
      </div>
    </DocumentTitle>
  )
}

export default connect(
  mapStateToProps,
)(Lock)
