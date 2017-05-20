// external imports
import * as React from 'react'
import { connect } from 'react-redux'
import ReactTooltip from 'react-tooltip'

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
      <button className="btn btn-primary" onClick={handleClick}>
        <span className="glyphicon glyphicon-floppy-disk"></span>
        Save
      </button>
    )
  } else {
    return (
      <div>
        <ReactTooltip id="saveButtonTooltip" place="bottom" type="error" effect="solid"/>
        <div data-for="saveButtonTooltip" data-tip={saveability.error}>
          <button className="btn btn-primary" disabled={true}>
            <span className="glyphicon glyphicon-floppy-disk"></span>
            Save
          </button>
        </div>
      </div>
    )
  }
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(SaveTemplate)
