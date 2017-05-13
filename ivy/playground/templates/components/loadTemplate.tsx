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
  handleChange: (id: string): void => {
    dispatch(loadTemplate(id))
  }
})

const LoadTemplate = ({ idList, selected, handleChange }) => {
  const options = idList.map(id => {
    return <option key={id} value={id}>{id}</option>
  })
  options.unshift(<option key={""} value={""}>Load Template...</option>)
  return (
    <select className="form-control" value={""} onChange={ (e) => handleChange(e.target.value) }>
      {options}
    </select>
  )
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(LoadTemplate)
