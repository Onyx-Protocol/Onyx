import React from 'react'
import { BaseList, TableList, EmptyContent } from 'features/shared/components'
import Item from  './ListItem'
import actions from 'actions'

const commonProps = {
  skipQuery: true,
  wrapperComponent: TableList,
  wrapperProps: {
    titles: ['Token ID'],
  }
}

export const ClientTokenList = BaseList.connect(
  (state, ownProps) =>
    BaseList.mapStateToProps('clientAccessToken', Item, {
      ...commonProps,
      label: 'Client access tokens',
      firstTimeContent:
        <EmptyContent>
          Client access tokens provide access to accounts, assets, transactions, and
          other objects on this core. If you're connecting to this core via localhost,
          you don't need an access token.
        </EmptyContent>
    })(state, ownProps),

  (dispatch) => ({
    ...BaseList.mapDispatchToProps('clientAccessToken')(dispatch),
    itemActions: {
      delete: (token) => {
        dispatch(actions.clientAccessToken.deleteItem(
          token.id,
          `Really delete access token ${token.id}?`,
          `Deleted access token ID ${token.id}.`
        ))
      }
    },
  })
)

export const NetworkTokenList = BaseList.connect(
  (state, ownProps) =>
    BaseList.mapStateToProps('networkAccessToken', Item, {
      ...commonProps,
      label: 'Network access tokens',
      firstTimeContent:
        <EmptyContent>
          Network access tokens allow other Chain Core instances to connect to this core.
        </EmptyContent>
    })(state, ownProps),

  (dispatch) => ({
    ...BaseList.mapDispatchToProps('networkAccessToken')(dispatch),
    itemActions: {
      delete: (token) => {
        dispatch(actions.networkAccessToken.deleteItem(
          token.id,
          `Really delete access token ${token.id}?`,
          `Deleted access token ID ${token.id}.`
        ))
      }
    },
  })
)
