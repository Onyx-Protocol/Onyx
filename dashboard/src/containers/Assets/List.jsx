import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Asset/Item'

import actions from '../../actions'
import { push } from 'react-router-redux'

const type = "asset"

const dispatch = (dispatch) => Object.assign({},
  mapDispatchToProps(type)(dispatch),
  {
    itemActions: {
      showCirculation: (id) => {
        let query = `asset_id='${id}'`
        dispatch(actions.balance.updateQuery(query))
        dispatch(actions.balance.resetPage())
        dispatch(push('/balances'))
      },
    }
  }
)


export default connect(
  mapStateToProps(type, Item),
  dispatch
)
