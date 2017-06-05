// external imports
import * as React from 'react'
import { connect } from 'react-redux'

// ivy imports
import { Input } from '../../inputs/types'
import { getWidget } from '../../contracts/components/parameters'
import { getItemMap as getAssetMap } from '../../assets/selectors'

// internal imports
import { getSpendContract, getClauseUnlockInput } from '../selectors'

const UnlockDestination = (props: { assetMap, contract, unlockInput: Input }) => {
  if (props.unlockInput === undefined || props.assetMap === undefined) {
    return <div></div>
  } else {
    return (
      <section>
        <h4>Unlocked Value Destination</h4>
        {getWidget("unlockValue.accountInput")}
        <div className="form-group">
          <div className="input-group">
            <div className="input-group-addon">Asset</div>
            <input type="text" className="form-control" value={props.assetMap[props.contract.assetId].alias} disabled />
          </div>
        </div>
        <div className="form-group">
          <div className="input-group">
            <div className="input-group-addon">Amount</div>
            <input type="text" className="form-control" value={props.contract.amount} disabled />
          </div>
        </div>
      </section>
    )
  }
}

export default connect(
  (state) => ({ assetMap: getAssetMap(state), unlockInput: getClauseUnlockInput(state), contract: getSpendContract(state) })
)(UnlockDestination)
