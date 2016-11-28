import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import Item from  './ListItem'
import actions from 'actions'

const clientType = 'client_access_token'
const networkType = 'network_access_token'

const emptyContentClient = <div className="emptyContainer">
  Client access tokens provide access to accounts, assets, transactions, and
  other objects on this core. If you're connecting to this core via localhost,
  you don't need an access token.
</div>

const emptyContentNetwork = <div className="emptyContainer">
  Network access tokens allow other Chain Core instances to connect to this core.
</div>

const stateToProps = (type, emptyListContent) => (state, ownProps) =>
  BaseList.mapStateToProps(type, Item, {
    skipQuery: true,
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Token ID'],
    },
    emptyContent: emptyListContent
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
  stateToProps(clientType, emptyContentClient),
  dispatchToProps(clientType)
)

export const NetworkTokenList = BaseList.connect(
  stateToProps(networkType, emptyContentNetwork),
  dispatchToProps(networkType),
)
