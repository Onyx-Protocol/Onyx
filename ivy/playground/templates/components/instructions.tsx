import * as React from 'react'
import { connect } from 'react-redux'

import app from '../../app'
import { isError, mustCompileTemplate } from '../util'
import { getTemplate } from '../selectors'
import { Item } from '../types'

const mapStateToProps = (state) => {
  const template = getTemplate(state)
  if (isError(template)) throw "uncaught compiler error"
  return { template }
}

const Instructions = ({ template }) => {
  const instructions = template.instructions.join(" ")
  return (
    <app.components.Section name="Compiled">
      <div className="codeblock">
        { instructions }
      </div>
    </app.components.Section>
  )
}

export default connect(
  mapStateToProps,
)(Instructions)
