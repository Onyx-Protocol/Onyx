import * as React from 'react'
import { connect } from 'react-redux'

import * as app from '../../app/types'
import { load } from '../actions'
import { getIdList, getTemplateState, getSelected } from '../selectors'
import { TemplateState } from '../types'

const mapStateToProps = (state: app.AppState) => {
  return {
    idList: getIdList(state),
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleChange: (id: string): void => {
    dispatch(load(id))
  }
})

const Load = ({ idList, selected, handleChange }) => {
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
)(Load)
