import * as React from 'react'
import { connect } from 'react-redux'
import { getTemplate, getItemMap } from '../selectors'
import { isError } from '../util'
import { saveTemplate } from '../actions'

function SaveUnconnected({ reset, disabled }) {
  return <button className="btn btn-primary" onClick={reset} disabled={disabled}>Save</button>
}

function mapStateToSaveProps(state) {
  let template = getTemplate(state)
  let templates = getItemMap(state)
  return {
    disabled: (isError(template) || templates[template.name] !== undefined)
  }
}

function mapDispatchToSaveProps(dispatch) {
  return {
    reset: () => {
      dispatch(saveTemplate())
    }
  }
}

const Save = connect(
  mapStateToSaveProps,
  mapDispatchToSaveProps
)(SaveUnconnected)

export default Save