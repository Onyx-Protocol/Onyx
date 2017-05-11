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
  let jsx = (<li><a href="#" onClick={handleClick}>Seed</a></li>)
  if (disable) {
    handleClick = () => {}
    jsx = (<li className='disabled'><a data-for="seedButtonTooltip" data-tip="Chain Core already has accounts and assets created." href="#" onClick={handleClick}>Seed</a></li>)
  }
  return jsx
}

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Seed)
