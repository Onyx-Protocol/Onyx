import * as React from 'react'
import { connect } from 'react-redux'

import { create } from '../actions'

import { isValid, getContractValue } from '../selectors'

import ReactTooltip from 'react-tooltip'

const mapStateToProps = (state) => {
  return {
    disabled: (isValid(state) === undefined) || (getContractValue(state) === undefined)
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleClick() {
    dispatch(create())
  }
})

type Props = {
  disabled: boolean,
  handleClick: (e) => undefined
}

const CreateFooter = (props: Props) => {
  let td = props.disabled ? 
    <td data-tip="One or more inputs is invalid."><button className="btn btn-primary btn-wide" disabled={true}>Create</button></td>
  :
    <td><button className="btn btn-primary btn-wide" onClick={props.handleClick}>Create</button></td>  
  return <div><ReactTooltip type="error" effect="solid"/><table><tbody><tr>{td}</tr></tbody></table></div>
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(CreateFooter)