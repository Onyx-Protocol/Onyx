import * as React from 'react'
import { connect } from 'react-redux'

import { getCompiled } from '../selectors'
import { Template } from '../types'

const ErrorAlert = (props: { compiled: Template }) => {
  return (
    <div className="panel-body inner">
      <h1>Compiled</h1>
      <div className="alert alert-danger" role="alert">
        <span className="glyphicon glyphicon-exclamation-sign" style={{marginRight: "5px"}}></span>
        <span className="sr-only">Error:</span>
        {props.compiled.error}
      </div>
    </div>
  )
}

const mapStateToProps = (state) => {
  return {
    compiled: getCompiled(state),
  }
}

export default connect(
  mapStateToProps
)(ErrorAlert)
