import { BaseList, TableList } from 'features/shared/components'
import Item from  './ListItem'
import actions from 'actions'

const clientType = 'client_access_token'
const networkType = 'network_access_token'

const stateToProps = (type) => (state, ownProps) =>
  BaseList.mapStateToProps(type, Item, {
    skipQuery: true,
    wrapperComponent: TableList,
    wrapperProps: {
        titles: ['Token ID']
    }
  })(state, ownProps)


const dispatchToProps = (type) => (dispatch) => {
  return {
    ...BaseList.mapDispatchToProps(type)(dispatch),
    itemActions: {
      delete: (token) => {
        dispatch(actions[type].deleteItem(
          token.id,
          `Really delete access token ${token.id}?`,
          `Deleted access token ID ${token.id}.`
        ))
      }
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
