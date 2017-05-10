import * as React from 'react'
import { connect } from 'react-redux'

import app from '../../app'
import { getOpcodes } from '../selectors'

const mapStateToProps = (state) => {
  const opcodes = getOpcodes(state)
  if (opcodes === "") throw "uncaught compiler error"
  return { opcodes }
}

const Instructions = ({ opcodes }) => {
  return (
    <div className="panel-body inner">
      <h1>Compiled</h1>
      <div className="codeblock instructions">
        { opcodes }
      </div>
    </div>
  )
}

export default connect(
  mapStateToProps,
)(Instructions)
