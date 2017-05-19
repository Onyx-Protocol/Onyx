// external imports
import * as React from 'react'
import { connect } from 'react-redux'

// ivy imports
import { create } from '../../contracts/actions'
import { getIsCalling } from '../../contracts/selectors'

// internal imports
import { getContractValue } from '../selectors'

const mapStateToProps = (state) => {
  return {
    isCalling: getIsCalling(state)
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleClick() {
    dispatch(create())
  }
})

type Props = {
  isCalling: boolean,
  handleClick: (e) => undefined
}

const LockButton = ({ isCalling, handleClick }: Props) => {
  const td = <td><button className="btn btn-primary btn-lg form-button" disabled={isCalling} onClick={handleClick}>Lock Value</button></td>
  return <div><table><tbody><tr>{td}</tr></tbody></table></div>
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(LockButton)
