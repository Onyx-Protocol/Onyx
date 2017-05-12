import * as React from 'react'
import { connect } from 'react-redux'
import { reset } from '../actions'

const mapStateToProps = undefined
const mapDispatchToProps = (dispatch) => ({
  handleClick() {
    dispatch(reset())
  }
})

const Reset = ({ handleClick }) => {
  return <li><a href="#" onClick={handleClick}>Reset</a></li>
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Reset)
