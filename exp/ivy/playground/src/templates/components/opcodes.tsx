import * as React from 'react'
import { connect } from 'react-redux'

import { getOpcodes } from '../selectors'

const mapStateToProps = (state) => {
  const opcodes = getOpcodes(state)
  if (opcodes === "") throw "uncaught compiler error"
  return { opcodes }
}

const Opcodes = ({ opcodes }) => {
  return (
    <div className="panel-body inner">
      <h1>Compiled</h1>
      <pre className="wrap">
        { opcodes }
      </pre>
    </div>
  )
}

export default connect(
  mapStateToProps,
)(Opcodes)
