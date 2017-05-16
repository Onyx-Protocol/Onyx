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
    <div className="dropdown">
  <button className="btn btn-primary dropdown-toggle" type="button" id="dropdownMenu1" data-toggle="dropdown" aria-haspopup="true" aria-expanded="true">
    <span className="glyphicon glyphicon-open"></span>
    Load Template
  </button>
  <ul className="dropdown-menu" aria-labelledby="dropdownMenu1">
    <li><a href="#">LockWithPublicKey</a></li>
    <li><a href="#">TradeOffer</a></li>
    <li><a href="#">Something else here</a></li>
    <li><a href="#">Separated link</a></li>
  </ul>
</div>
  )
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(LoadTemplate)
