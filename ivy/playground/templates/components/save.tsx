import * as React from 'react'
import { connect } from 'react-redux'
import { getSaveability } from '../selectors'
import { saveTemplate } from '../actions'

function SaveUnconnected({ reset, saveability }) {
  if (saveability.saveable) {
    return (
      <td>
        <button className="btn btn-primary" onClick={reset}>
          <span className="glyphicon glyphicon-floppy-disk"></span>
          Save
        </button>
      </td>
    )
  } else {
    return (
      <td data-for="saveButtonTooltip" data-tip={saveability.error}>
        <button className="btn btn-primary" disabled={true}>
          <span className="glyphicon glyphicon-floppy-disk"></span>
          Save
        </button>
     </td>
    )
  }
}

function mapStateToSaveProps(state) {
  let saveability = getSaveability(state)
  return {
    saveability
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
