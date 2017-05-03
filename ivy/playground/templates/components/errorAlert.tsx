import * as React from 'react'
import { connect } from 'react-redux'

import { getSelectedTemplate } from '../selectors'

const ErrorAlert = ({ errorMessage }) => {
  return (
    <div className="row">
      <div className="col-xs-8" style={{marginTop: 20}}>
      <div className="alert alert-danger" role="alert">
        <span className="glyphicon glyphicon-exclamation-sign" style={{marginRight: "5px"}}></span>
        <span className="sr-only">Error:</span>
          {errorMessage}
        </div>
      </div>
    </div>
  )
}

const mapStateToProps = (state) => {
  return {
    template: getSelectedTemplate(state),
  }
}

export default connect(
  mapStateToProps
)(ErrorAlert)
