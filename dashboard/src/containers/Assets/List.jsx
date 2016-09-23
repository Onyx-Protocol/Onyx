import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Asset/Item'

import actions from '../../actions'
import { push } from 'react-router-redux'

const type = 'asset'

const dispatch = (dispatch) => Object.assign({},
  mapDispatchToProps(type)(dispatch),
  {
    itemActions: {
      showCirculation: (item) => {
        let query = `asset_id='${item.id}'`
        if (item.alias) {
          query = `asset_alias='${item.alias}'`
        }

        dispatch(actions.balance.updateQuery(query))
        dispatch(push('/balances'))
      },
    }
  }
)


export default connect(
  mapStateToProps(type, Item),
  dispatch
)
