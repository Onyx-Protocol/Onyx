import * as React from 'react'
import { connect } from 'react-redux'
import { reset } from '../actions'

const mapStateToProps = undefined
const mapDispatchToProps = (dispatch) => ({
  handleClick(e) {
    e.preventDefault()
    let confirm = window.confirm("Are you sure you want to reset the playground? Your contracts and any data used to generate them will be forgotten, making it impossible to recover any locked assets.");
    if (confirm) {
      dispatch(reset())
    }
  }
})

const Reset = ({ handleClick }) => {
  return <li><a href="#" onClick={(e) => handleClick(e)}>Reset</a></li>
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Reset)
