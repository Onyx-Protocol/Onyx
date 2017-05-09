import * as React from 'react'
import { connect } from 'react-redux'
import { getCompiled, getItemMap } from '../selectors'
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
  let compiled = getCompiled(state)
  let templates = getItemMap(state)
  return {
    error: (!compiled || compiled.error !== ""),
    exists: compiled && compiled.error === "" && templates[compiled.name] !== undefined
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