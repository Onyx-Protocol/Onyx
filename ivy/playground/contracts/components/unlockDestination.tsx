// external imports
import * as React from 'react'
import { connect } from 'react-redux'

// ivy imports
import { Input } from '../../inputs/types'
import { getWidget } from '../../contracts/components/parameters'

// internal imports
import { getClauseUnlockInput } from '../selectors'

const UnlockDestination = (props: { unlockInput: Input }) => {
  if (props.unlockInput === undefined) {
    return <div></div>
  } else {
    return (
      <section>
        <h4>Unlocked Value Destination</h4>
        {getWidget("unlockValue.accountInput")}
      </section>
    )
  }
}

export default connect(
  (state) => ({ unlockInput: getClauseUnlockInput(state) })
)(UnlockDestination)
