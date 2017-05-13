import * as React from 'react'
// external imports
import { connect } from 'react-redux'

// internal imports
import { saveTemplate } from '../actions'
import { getSaveability } from '../selectors'

const mapStateToProps = (state) => {
  let saveability = getSaveability(state)
  return {
    saveability
  }
}

const mapDispatchToProps = (dispatch) => {
  return {
    handleClick() {
      dispatch(saveTemplate())
    }
  }
}

const SaveTemplate = ({ handleClick, saveability }) => {
  if (saveability.saveable) {
    return (
      <td>
        <button className="btn btn-primary" onClick={handleClick}>
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

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(SaveTemplate)
