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
    <app.components.Section name="Compiled">
      <div className="codeblock">
        { opcodes }
      </div>
    </app.components.Section>
  )
}

export default connect(
  mapStateToProps,
)(Instructions)
