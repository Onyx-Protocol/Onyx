import * as React from 'react'
import { connect } from 'react-redux'

import { getCompiled } from '../selectors'
import { CompilerResult } from '../types'

const ErrorAlert = (props: { compiled: CompilerResult }) => {
  return (
    <div className="row">
      <div className="col-xs-8" style={{marginTop: 20}}>
      <div className="alert alert-danger" role="alert">
        <span className="glyphicon glyphicon-exclamation-sign" style={{marginRight: "5px"}}></span>
        <span className="sr-only">Error:</span>
          {props.compiled.error}
        </div>
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
