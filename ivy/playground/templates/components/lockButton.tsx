import * as React from 'react'
import { connect } from 'react-redux'

import { create } from '../../contracts/actions'

import { getCreateability, getContractValue } from '../selectors'

import ReactTooltip from 'react-tooltip'

const mapStateToProps = (state) => {
  return {
    createability: getCreateability(state)
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleClick() {
    dispatch(create())
  }
})

type Props = {
  createability: { createable: boolean, error: string },
  handleClick: (e) => undefined
}

const CreateButton = ({ createability, handleClick }: Props) => {
  let td = createability.createable ?
    <td><button className="btn btn-primary btn-lg" onClick={handleClick}>Lock Value</button></td>
  :
    <td data-for="createButtonTooltip" data-tip={createability.error}><button className="btn btn-primary btn-lg" disabled={true}>Lock Value</button></td>
  return <div><ReactTooltip id="createButtonTooltip" type="error" effect="solid"/><table><tbody><tr>{td}</tr></tbody></table></div>
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(CreateButton)
