// external imports
import * as React from 'react'
import { connect } from 'react-redux'
import DocumentTitle from 'react-document-title'
import Modal from 'react-modal'

// ivy imports
import Section from '../../app/components/section'
import { Display } from '../../contracts/components/display'
import { ContractParameters, ContractValue } from '../../contracts/components/parameters'

// internal imports
import Editor from './editor'
import LockButton from './lockButton'
import { getError, getSource, getContractParameters, getCompiled } from '../selectors'
import { closeModal } from '../actions'

const mapStateToProps = (state) => {
  const source = getSource(state)
  const contractParameters = getContractParameters(state)
  const instantiable = contractParameters && contractParameters.length > 0
  const error = getError(state)
  return { source, instantiable, error }
}

const mapDispatchToProps = (dispatch) => {
  return {
    closeModal: () => {
      console.log("closing")
      dispatch(closeModal())
    }
  }
}

const ErrorAlert = (props: { error: string }) => {
  return (
    <div style={{margin: '25px 0 0 0'}} className="alert alert-danger" role="alert">
      <span className="sr-only">Error:</span>
      <span className="glyphicon glyphicon-exclamation-sign" style={{marginRight: "5px"}}></span>
      {props.error}
    </div>
  )
}

const Lock = ({ source, instantiable, error, closeModal }) => {

  let modal = <Modal
    isOpen={true}
    contentLabel="Welcome to the Ivy Playground"
    onRequestClose={() => { console.log("requestedClose") }}>

    <h2>Welcome to the Ivy Playground</h2>
    <button>close</button>
    <div>Welcome to the Ivy Playground

    We've seeded your Chain Core with a few accounts and assets to help you get started. You can create more by visiting the Dashboard.

    The tutorial is a great place to start, which you can visit any time by clicking the link in the top right. Enjoy!</div>
    <form>
      <input />
      <button>Let's get started</button>
    </form>
  </Modal>

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
          <div className="form-wrapper">
            { error ? <ErrorAlert error={error} /> : <div/> }
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
        {modal}
        <Editor />
        {instantiate}
      </div>
    </DocumentTitle>
  )
}

export default connect(
  mapStateToProps, mapDispatchToProps
)(Lock)
