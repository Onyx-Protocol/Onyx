//external imports
import * as React from 'react'
import { connect } from 'react-redux'

// internal imports
import { loadTemplate } from '../actions'
import { INITIAL_ID_LIST } from '../constants'

const mapDispatchToProps = (dispatch) => ({
  handleClick() {
    dispatch(loadTemplate(INITIAL_ID_LIST[0]))
  }
})

const NewTemplate = ({ handleClick }) => {
  return (
    <div className="dropdown">
      <button onClick={handleClick} className="btn btn-primary dropdown-toggle" type="button" id="dropdownMenu1" data-toggle="dropdown" aria-haspopup="true" aria-expanded="true">
        <span className="glyphicon glyphicon-plus"></span>
        New
      </button>
    </div>
  )
}

export default connect(
  undefined,
  mapDispatchToProps
)(NewTemplate)
