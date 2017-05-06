import * as React from 'react'
import { connect } from 'react-redux'
import DocumentTitle from 'react-document-title'

import app from '../../app'
import { Item as Template } from '../../templates/types'
import CreateFooter from './createfooter'

import { getSelectedTemplate, getState as getContractsState } from '../selectors'

import { ContractParameters } from './parameters'

import Select from './select'

import { Display } from './display' 

const mapStateToProps = (state) => {
  const template = getSelectedTemplate(state)
  const contracts = getContractsState(state)
  return { template, contracts }
}

const Create = (props) => {
  console.log(props.contracts.inputMap)
  if (Object.keys(props.contracts.inputMap).length === 0) {
    // TODO(boymanjor): Fix me if possible
    return <div />
  }
  return (
    <DocumentTitle title='Create Contract'>
      <app.components.Section name="Create Contract" footer={<CreateFooter />}>
        <div className="form-wrapper">
          <section>
            <h4>Select Contract Template</h4>
            <div className="form-group">
              <Select />
            </div>
            <div>
              <Display source={props.template.source} />
            </div>
          </section>
          <ContractParameters />
        </div>
      </app.components.Section>
    </DocumentTitle>
  )
}

export default connect(
  mapStateToProps,
)(Create)


