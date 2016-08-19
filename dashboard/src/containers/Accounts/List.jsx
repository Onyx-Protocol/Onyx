import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Account/Item'

import actions from '../../actions'
import { push } from 'react-router-redux'

const type = "account"

const dispatch = (dispatch) => Object.assign({},
  mapDispatchToProps(type)(dispatch),
  {
    itemActions: {
      showTransactions: (id) => {
        let query = `inputs(account_id='${id}') OR outputs(account_id='${id}')`
        dispatch(actions.transaction.updateQuery(query))
        dispatch(actions.transaction.resetPage())
        dispatch(push('/transactions'))
      },
      showBalances: (id) => {
        let query = `account_id='${id}' AND asset_id=$1`
        dispatch(actions.balance.updateQuery(query))
        dispatch(actions.balance.resetPage())
        dispatch(push('/balances'))
      }
    }
  }
)

export default connect(
  mapStateToProps(type, Item),
  dispatch
)
