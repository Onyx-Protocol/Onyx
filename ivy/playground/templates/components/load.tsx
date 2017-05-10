import * as React from 'react'
import { connect } from 'react-redux'

import * as app from '../../app/types'
import { load } from '../actions'
import { getIdList, getState, getSelectedTemplate } from '../selectors'
import { State } from '../types'

const mapStateToProps = (state: app.AppState) => {
  return {
    idList: getIdList(state),
    selected: getSelectedTemplate(state)
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleChange: (id: string): void => {
    dispatch(load(id))
  }
})

const Load = ({ selected, idList, handleChange }) => {
  const options = idList.map(id => {
    return <option key={id} value={id}>{id}</option>
  })
  options.unshift(<option key={""} value={""}>Load Template...</option>)
  return (
    <select value={selected} className="form-control" onChange={ (e) => handleChange(e.target.value) }>
      {options}
    </select>
  )
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Load)
