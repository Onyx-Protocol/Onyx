import * as React from 'react'
import { connect } from 'react-redux'
import { seed } from '../actions'

const mapStateToProps = (state) => {
  return {
    disable: state.accounts.idList.length !== 0 || state.assets.idList.length !== 0
  }
}

const mapDispatchToProps = (dispatch) => ({
  handleClick() {
    dispatch(seed())
  }
})

const Seed = ({ handleClick, disable }) => {
  let jsx = (<a href="#" onClick={handleClick}>Seed</a>)
  if (disable) {
    handleClick = () => {}
    jsx = (<a className='disabled' data-for="seedButtonTooltip" data-tip="Chain Core already has accounts and assets created." href="#" onClick={handleClick}>Seed</a>)
  }
  return jsx
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Seed)
