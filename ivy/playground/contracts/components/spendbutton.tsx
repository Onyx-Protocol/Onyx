import * as React from 'react'
import { connect } from 'react-redux'
import { spend } from '../actions'
import { areSpendInputsValid } from '../selectors'

const mapDispatchToProps = (dispatch) => ({
  handleSpendClick() {
    dispatch(spend())
  }
})
const SpendButton = (props: {enabled: boolean, handleSpendClick: (e)=>undefined} ) => {
  return <button className="btn btn-primary btn-lg" disabled={!props.enabled} onClick={props.handleSpendClick}>Spend Contract</button>
}

export default connect(
  (state) => ({ enabled: areSpendInputsValid(state) }),
  mapDispatchToProps
)(SpendButton)
