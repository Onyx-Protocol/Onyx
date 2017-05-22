// external imports
import * as React from 'react'
import { connect } from 'react-redux'
import DocumentTitle from 'react-document-title'

// ivy imports
import Section from '../../app/components/section'
import { Display } from '../../contracts/components/display'
import { ContractParameters, ContractValue } from '../../contracts/components/parameters'

// internal imports
import Editor from './editor'
import LockButton from './lockButton'
import { getLockError, getSource, getContractParameters, getCompiled } from '../selectors'

const mapStateToProps = (state) => {
  const source = getSource(state)
  const compiled = getCompiled(state)
  const instantiable = compiled && compiled.error === ''
  const contractParameters = getContractParameters(state)
  const hasParams = contractParameters && contractParameters.length > 0
  const error = getLockError(state)
  return { source, instantiable, hasParams, error }
}

const ErrorAlert = (props: { error: string }) => {
  let jsx = <small />
  if (props.error) {
    jsx = (
      <div style={{margin: '25px 0'}} className="alert alert-danger" role="alert">
        <span className="sr-only">Error:</span>
        <span className="glyphicon glyphicon-exclamation-sign" style={{marginRight: "5px"}}></span>
        {props.error}
      </div>
    )
  }
  return jsx
}

const Lock = ({ source, instantiable, hasParams, error }) => {
  let instantiate
  let contractParams
  if (instantiable) {
    contractParams = <div />
    if (hasParams) {
      contractParams = (
        <Section name="Contract Arguments">
          <div className="form-wrapper">
            <ContractParameters />
          </div>
          <div className="form-wrapper">
          </div>
        </Section>
      )
    }
    instantiate = (
      <div>
        <Section name="Value to Lock">
          <div className="form-wrapper">
            <ContractValue />
          </div>
        </Section>
        {contractParams}
        <ErrorAlert error={error} />
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
