import * as React from 'react'
import { connect } from 'react-redux'
import DocumentTitle from 'react-document-title'

import app from '../../app'
import { Template } from '../types'
import LockButton from './lockButton'

import { getSource, getContractParameters, getCompiled } from '../selectors'

import { ContractParameters, ContractValue } from '../../contracts/components/parameters'

import Editor from './editor'

import { Display } from '../../contracts/components/display'

const mapStateToProps = (state) => {
  const source = getSource(state)
  const contractParameters = getContractParameters(state)
  const compiled = getCompiled(state)
  let instantiable = (contractParameters !== undefined) && (compiled !== undefined)
  return { source, instantiable }
}

const Lock = ({ source, instantiable }) => {
  let instantiate
  if (instantiable) {
    instantiate = (
      <div>
        <app.components.Section name="Contract Value">
          <div className="form-wrapper">
            <ContractValue />
          </div>
        </app.components.Section>
        <app.components.Section name="Contract Arguments">
          <div className="form-wrapper">
            <ContractParameters />
          </div>
        </app.components.Section>
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
