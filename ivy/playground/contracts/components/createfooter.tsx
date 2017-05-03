import * as React from 'react'
import { connect } from 'react-redux'

import { create } from '../actions'

import { isValid, getContractValue } from '../selectors'

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
  return <button className="btn btn-primary btn-wide" disabled={props.disabled} onClick={props.handleClick}>Create</button>
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(CreateFooter)