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
        dispatch(push('/transactions'))
      },
      showBalances: (id) => {
        let query = `account_id='${id}'`
        dispatch(actions.balance.updateQuery({
          query: query,
          sumBy: 'asset_id'
        }))
        dispatch(push('/balances'))
      },
      createControlProgram: (data) => dispatch(actions[type].createControlProgram(data))
    }
  }
)

export default connect(
  mapStateToProps(type, Item),
  dispatch
)
