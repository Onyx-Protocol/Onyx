// external imports
import * as React from 'react'
import { connect } from 'react-redux'
import ReactTooltip from 'react-tooltip'

// ivy imports
import { create } from '../../contracts/actions'
import { getIsCalling } from '../../contracts/selectors'

// internal imports
import { getCreateability, getContractValue } from '../selectors'

const mapStateToProps = (state) => {
  return {
    isCalling: getIsCalling(state),
    createability: getCreateability(state)
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleClick() {
    dispatch(create())
  }
})

type Props = {
  isCalling: boolean,
  createability: { createable: boolean, error: string },
  handleClick: (e) => undefined
}

const LockButton = ({ isCalling, createability, handleClick }: Props) => {
  let td
  if (isCalling) {
    td = <td data-for="createButtonTooltip" data-tip={'Attempting to lock value.'}><button className="btn btn-primary btn-lg form-button" disabled={true}>Lock Value</button></td>
  } else if (createability.createable) {
    td = <td><button className="btn btn-primary btn-lg form-button" onClick={handleClick}>Lock Value</button></td>
  } else {
    td = <td data-for="createButtonTooltip" data-tip={createability.error}><button className="btn btn-primary btn-lg form-button" disabled={true}>Lock Value</button></td>
  }
  return <div><ReactTooltip id="createButtonTooltip" type="error" effect="solid"/><table><tbody><tr>{td}</tr></tbody></table></div>
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(LockButton)
