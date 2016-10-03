import React from 'react'
import { BaseList } from 'features/shared/components'
import ItemList from 'components/ItemList'
import Item from  './ListItem'
import actions from 'actions'
import styles from './List.scss'


class List extends React.Component {
  render() {
    return (
      <ItemList {...this.props}>
        <div className={styles.toggleContainer}>
          <div className={styles.toggleInstructions}>
            <strong>Status:</strong>

            {!this.props.enabled && <span className='label label-danger'>DISABLED</span>}
            {this.props.enabled && <span className='label label-success'>ENABLED</span>}

            {this.props.description}
          </div>
          <div className={styles.toggleControl}>
            {!this.props.enabled && <button className='btn btn-success' onClick={() => this.props.confirmEnable(this.props.confirmEnableContent)}>
              Enable
            </button>}
            {this.props.enabled && <button className='btn btn-danger' onClick={() => this.props.confirmDisable(this.props.confirmDisableContent)}>
              Disable
            </button>}
          </div>
        </div>
      </ItemList>
    )
  }
}

const clientType = 'client_access_token'
const networkType = 'network_access_token'

const stateToProps = (type, additionalProps = {}) => (state) => {
  const enabledType = (type == clientType) ? 'requireClientToken' : 'requireNetworkToken'

  return {
    ...BaseList.mapStateToProps(type, Item)(state),
    ...additionalProps,
    enabled: state.core[enabledType],
    skipQuery: true,
  }
}

const dispatchToProps = (type) => (dispatch) => {
  const enableTokens = actions[type].enable
  const disableTokens = actions[type].disable
  const hideModal = actions.app.hideModal()

  return {
    ...BaseList.mapDispatchToProps(type)(dispatch),
    confirmEnable: (body) => dispatch(actions.app.showModal(body, enableTokens, hideModal)),
    confirmDisable: (body) => dispatch(actions.app.showModal(body, disableTokens, hideModal)),
    itemActions: {
      delete: (id) => dispatch(actions[type].deleteItem(id))
    },
  }
}

export const ClientTokenList = BaseList.connect(
  stateToProps(clientType, {
    description: <p>
      All applications (including the dashboard) will require a client token
      to access transactions, accounts, assets and core configuration
    </p>,
    confirmEnableContent: <p>
      Are you sure you want to require client access tokens for all apps?
      You will be logged out and asked to provide a valid client token immediately.
    </p>,
    confirmDisableContent: <p>Are you sure you want to allow insecure client access?</p>
  }),
  dispatchToProps(clientType),
  List
)

export const NetworkTokenList = BaseList.connect(
  stateToProps(networkType, {
    description: <p>
      All blockchain cores wil require a network token to sign, submit and
      synchronize transactions and blocks
    </p>,
    confirmEnableContent: <p>Are you sure you want to require network access tokens?</p>,
    confirmDisableContent: <p>Are you sure you want to allow insecure network API access?</p>
  }),
  dispatchToProps(networkType),
  List
)
