// external imports
import * as React from 'react'
import { connect } from 'react-redux'
import ReactTooltip from 'react-tooltip'

// internal imports
import Ace from './ace'
import ErrorAlert from './errorAlert'
import LoadTemplate from './loadTemplate'
import SaveTemplate from './saveTemplate'
import Opcodes from './opcodes'
import { getCompiled, getSource } from '../selectors'

// Handles syntax highlighting
require('../util/ivymode.js')

const mapStateToProps = (state) => {
  return {
    compiled: getCompiled(state),
    source: getSource(state)
  }
}

const Editor = ({ compiled, source }) => {
  return (
    <div>
      <ReactTooltip id="saveButtonTooltip" place="bottom" type="error" effect="solid"/>
      <div className="panel panel-default">
        <div className="panel-heading clearfix">
          <h1 className="panel-title pull-left">Contract Template</h1>
          <table className="pull-right">
            <tbody>
              <tr>
                <td style={{width: 300, paddingRight: 10}}><LoadTemplate /></td>
                <SaveTemplate />{/* SaveTemplate returns a <td> element */}
              </tr>
            </tbody>
          </table>
        </div>
        <Ace source={source} />
        { compiled && compiled.error !== "" ? <ErrorAlert errorMessage={compiled.error} /> : <Opcodes />}
      </div>
    </div>
  )
}

export default connect(
  mapStateToProps,
)(Editor)
