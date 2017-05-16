// external imports
import * as React from 'react'
import { connect } from 'react-redux'

// ivy imports
import { AppState } from '../../app/types'

// internal imports
import { loadTemplate } from '../actions'
import { getTemplateIds } from '../selectors'

const mapStateToProps = (state: AppState) => {
  return {
    idList: getTemplateIds(state),
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleClick: (e, id: string): void => {
    e.preventDefault()
    dispatch(loadTemplate(id))
  }
})

const LoadTemplate = ({ idList, selected, handleClick }) => {
  const options = idList.map(id => {
    return <li key={id}><a onClick={(e) => handleClick(e, id)} href='#'>{id}</a></li>
  })
  return (
    <div className="dropdown">
      <button className="btn btn-primary dropdown-toggle" type="button" id="dropdownMenu1" data-toggle="dropdown" aria-haspopup="true" aria-expanded="true">
        <span className="glyphicon glyphicon-open"></span>
        Load Template
      </button>
      <ul className="dropdown-menu" aria-labelledby="dropdownMenu1">
        {options}
      </ul>
    </div>
  )
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(LoadTemplate)
