import * as React from 'react'
import { connect } from 'react-redux'
import { spend } from '../actions'
import { areSpendInputsValid } from '../selectors'

const mapDispatchToProps = (dispatch) => ({
  handleSpendClick() {
    dispatch(spend())
  }
})

const UnlockButton = (props: {enabled: boolean, handleSpendClick: (e)=>undefined} ) => {
  return <button className="btn btn-primary btn-lg" disabled={!props.enabled} onClick={props.handleSpendClick}>Unlock Value</button>
}

export default connect(
  (state) => ({ enabled: areSpendInputsValid(state) }),
  mapDispatchToProps
)(UnlockButton)
