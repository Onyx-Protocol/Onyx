import React from 'react'
import { BaseList } from 'features/shared/components'
import ItemList from 'components/ItemList'
import Item from  './ListItem'
import actions from 'actions'
import styles from './List.scss'

const clientType = 'client_access_token'
const networkType = 'network_access_token'

const stateToProps = (type) => (state) => {
  return {
    ...BaseList.mapStateToProps(type, Item)(state),
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
  dispatchToProps(clientType),
  ItemList
)

export const NetworkTokenList = BaseList.connect(
  stateToProps(networkType),
  dispatchToProps(networkType),
  ItemList
)
