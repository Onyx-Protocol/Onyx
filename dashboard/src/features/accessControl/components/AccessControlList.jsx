import React from 'react'
import GrantListItem from './GrantListItem'
import { connect } from 'react-redux'
import { TableList, PageTitle, PageContent } from 'features/shared/components'
import { push, replace } from 'react-router-redux'
import actions from 'features/accessControl/actions'
import componentClassNames from 'utility/componentClassNames'
import styles from './AccessControlList.scss'

class AccessControlList extends React.Component {
  render() {
    const itemProps = {
      beginEditing: this.props.beginEditing,
      delete: this.props.delete,
    }
    const tokenList = <TableList titles={['Token Name', 'Policies']}>
      {this.props.tokens.map(item => <GrantListItem key={item.id} item={item} {...itemProps} />)}
    </TableList>

    const certList = <TableList titles={['Certificate', 'Policies']}>
      {this.props.certs.map(item => <GrantListItem key={item.id} item={item} {...itemProps} />)}
    </TableList>

    return (<div className={componentClassNames(this)}>
      <PageTitle title='Access control' />

      <PageContent>
        <div className={`btn-group ${styles.btnGroup}`} role='group'>
          <button
            className={`btn btn-default ${styles.btn} ${this.props.tokensButtonStyle}`}
            onClick={this.props.showTokens}>
              Tokens
          </button>

          <button
            className={`btn btn-default ${styles.btn} ${this.props.certificatesButtonStyle}`}
            onClick={this.props.showCertificates}>
              Certificates
          </button>
        </div>

        {this.props.tokensSelected && <div>
          <button
            className={`NewTokenButton btn btn-primary ${styles.newBtn}`}
            onClick={this.props.showTokenCreate}>
              + New token
          </button>

          {tokenList}
        </div>}

        {this.props.certificatesSelected && <div>
          <button
            className={`btn btn-primary ${styles.newBtn}`}
            onClick={this.props.showAddCertificate}>
              + Register certificate
          </button>

          {certList}
        </div>}
      </PageContent>
    </div>)
  }
}

const mapStateToProps = (state, ownProps) => {
  const items = state.accessControl.ids.map(id => state.accessControl.items[id])
  const tokensSelected = ownProps.location.query.type == 'token'
  const certificatesSelected = ownProps.location.query.type != 'token'

  return {
    tokens: items.filter(item => item.guardType == 'access_token'),
    certs: items.filter(item => item.guardType == 'x509'),
    tokensSelected,
    certificatesSelected,
    tokensButtonStyle: tokensSelected && styles.active,
    certificatesButtonStyle: certificatesSelected && styles.active,
  }
}

const mapDispatchToProps = (dispatch) => ({
  delete: (grant) => dispatch(actions.deleteToken(grant)),
  showTokens: () => dispatch(replace('/access-control?type=token')),
  showCertificates: () => dispatch(replace('/access-control?type=certificate')),
  showTokenCreate: () => dispatch(push('/access-control/create-token')),
  showAddCertificate: () => dispatch(push('/access-control/add-certificate')),
  beginEditing: (id) => dispatch(actions.beginEditing(id)),
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(AccessControlList)
