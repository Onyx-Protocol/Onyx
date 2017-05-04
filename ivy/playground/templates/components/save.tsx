import * as React from 'react'
import { connect } from 'react-redux'
import { getTemplate, getItemMap } from '../selectors'
import { isError } from '../util'
import { saveTemplate } from '../actions'

function SaveUnconnected({ reset, error, exists }) {
  if (error) {
    return <td data-tip="Contract template does not compile."><button className="btn btn-primary" disabled={true}>
        <span className="glyphicon glyphicon-floppy-disk"></span>
        Save
    </button></td>
  } else if (exists) {
    return <td data-tip="There is already a contract template with that name."><button className="btn btn-primary" disabled={true}>
        <span className="glyphicon glyphicon-floppy-disk"></span>
        Save
    </button></td>
  } else {
    return <td><button className="btn btn-primary" onClick={reset}>
      <span className="glyphicon glyphicon-floppy-disk"></span>
      Save
    </button></td>
  }
}

function mapStateToSaveProps(state) {
  let template = getTemplate(state)
  let templates = getItemMap(state)
  return {
    error: (isError(template)),
    exists: !isError(template) && templates[template.name] !== undefined
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