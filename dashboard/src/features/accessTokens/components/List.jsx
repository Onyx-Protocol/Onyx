import { BaseList } from 'features/shared/components'
import Item from  './ListItem'
import actions from 'actions'

const clientType = 'client_access_token'
const networkType = 'network_access_token'

const stateToProps = (type) => (state, ownProps) => {
  return {
    ...BaseList.mapStateToProps(type, Item)(state, ownProps),
    skipQuery: true,
  }
}

const dispatchToProps = (type) => (dispatch) => {
  return {
    ...BaseList.mapDispatchToProps(type)(dispatch),
    itemActions: {
      delete: (id) => dispatch(actions[type].deleteItem(id))
    },
  }
}

export const ClientTokenList = BaseList.connect(
  stateToProps(clientType),
  dispatchToProps(clientType)
)

export const NetworkTokenList = BaseList.connect(
  stateToProps(networkType),
  dispatchToProps(networkType)
)
