import * as React from 'react'
import { connect } from 'react-redux'
import { spend } from '../actions'

const mapDispatchToProps = (dispatch) => ({
  handleSpendClick() {
    dispatch(spend())
  }
})
const SpendFooter = (props: {enabled: boolean, handleSpendClick: (e)=>undefined} ) => {
  return <button className="btn btn-primary" disabled={!props.enabled} onClick={props.handleSpendClick}>Spend</button>
}

export default connect(
  (state) => ({ enabled: true }), // dummy! was (getResult(state) === "Contract can be spent with these inputs"))
  mapDispatchToProps
)(SpendFooter)