import { connect } from 'react-redux'
import { context } from 'utility/environment'
import chain from '_chain'
import { PageContent, ErrorBanner, PageTitle } from 'features/shared/components'
import React from 'react'
import styles from './Index.scss'
import testnetUtils from 'features/testnet/utils'

class Index extends React.Component {
  constructor(props) {
    super(props)
    this.state = {}
    this.deleteClick = this.deleteClick.bind(this)
  }

  deleteClick() {
    if (!window.confirm('Are you sure you want to delete all data on this core?')) {
      return
    }

    this.setState({deleteDisabled: true})

    chain.Core.reset(context()).then(() => {
      // TODO: Use Redux state reset and nav action instead of window.location.
      // Also, move confirmation message to a bonafide flash div. alert() in a
      // browser microtask is going away. cf https://www.chromestatus.com/features/5647113010544640
      setTimeout(function(){
        window.location.href = '/'
      }, 500)
    }).catch((err) => {
      this.setState({
        deleteError: err,
        deleteDisabled: false,
      })
    })
  }

  render() {
    const {
      onTestnet,
      testnetBlockchainMismatch,
      testnetNetworkMismatch,
      testnetNextReset,
    } = this.props

    let generatorUrl
    if (this.props.core.generator) {
      generatorUrl = window.location.origin
    } else if (onTestnet) {
      generatorUrl = <span>
        {this.props.core.generatorUrl}
        &nbsp;
        <span className='label label-primary'>Chain Testnet</span>
      </span>
    } else {
      generatorUrl = this.props.core.generatorUrl
    }

    let configBlock = (
      <div className={`${styles.left} ${styles.col}`}>
        <div>
          <h4>Configuration</h4>
          <table className={styles.table}>
            <tbody>
              <tr>
                <td className={styles.row_label}>Core type:</td>
                <td>{this.props.core.coreType}</td>
              </tr>
              <tr>
                <td className={styles.row_label}>Setup time:</td>
                <td>{this.props.core.configuredAt}</td>
              </tr>
              <tr>
                <td className={styles.row_label}>Version:</td>
                <td><code>{this.props.core.version}</code></td>
              </tr>
              <tr>
                <td colSpan={2}><hr /></td>
              </tr>
              <tr>
                <td className={styles.row_label}>Generator URL:</td>
                <td>{generatorUrl}</td>
              </tr>
              {onTestnet && !!testnetNextReset &&
                <tr>
                  <td className={styles.row_label}>Next Chain Testnet data reset:</td>
                  <td>{testnetNextReset.toString()}</td>
                </tr>}
              {!this.props.core.generator &&
                <tr>
                  <td className={styles.row_label}>Network Access Token:</td>
                  <td><code>{this.props.core.generatorAccessToken}</code></td>
                </tr>}
              <tr>
                <td className={styles.row_label}>Blockchain ID:</td>
                <td><code className={styles.block_hash}>{this.props.core.blockchainId}</code></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    )

    let testnetErr
    if (onTestnet) {
      if (testnetBlockchainMismatch) {
        testnetErr = 'Chain Testnet has been reset. Please reset your core below.'
      } else if (testnetNetworkMismatch) {
        testnetErr = {message: <span>This core is no longer compatible with Chain Testnet. <a href='https://chain.com/docs' target='_blank'>Please upgrade Chain Core</a>.</span>}
      }
    }

    let networkStatusBlock = (
      <div className={`${styles.right} ${styles.col}`}>
        <div>
          <h4>Network status</h4>

          <table className={styles.table}>
            <tbody>
              <tr>
                <td className={styles.row_label}>Generator block:</td>
                <td className={styles.row_value}>{this.props.core.generatorBlockHeight}</td>
              </tr>
              <tr>
                <td className={styles.row_label}>Local block:</td>
                <td className={styles.row_value}>{this.props.core.blockHeight}</td>
              </tr>
              <tr>
                <td className={styles.row_label}>Replication lag:</td>
                <td className={`${styles.replication_lag} ${styles[this.props.core.replicationLagClass]}`}>
                  {this.props.core.replicationLag === null ? '???' : this.props.core.replicationLag}
                </td>
              </tr>
            </tbody>
          </table>

          {testnetErr && <ErrorBanner title='Chain Testnet error' error={testnetErr} />}
        </div>
      </div>
    )

    let resetDataBlock = (
      <div className='row'>
        <div className='col-sm-6'>
          <h4>Reset data</h4>

          {this.props.core.production ?
            <p>
              This core is configured to run in production. Production
              blockchains cannot be reset.
            </p> :
            <div>
              <p>
                This will permanently delete all data stored in this core,
                including blockchain data, accounts, assets, indexes,
                and MockHSM keys.
              </p>

              {this.state.deleteError && <ErrorBanner
                title='Error resetting data'
                message={this.state.deleteError.toString()}
              />}

              <button
                className='btn btn-danger'
                onClick={this.deleteClick}
                disabled={this.state.deleteDisabled}
              >
                Delete all data
              </button>
            </div>}
        </div>
      </div>
    )

    return (
      <div className={`flex-container ${styles.mainContainer}`}>
        <PageTitle title='Core' />

        <PageContent>
          <div className={`${styles.top} ${styles.flex}`}>
            {configBlock}
            {networkStatusBlock}
          </div>

          {resetDataBlock}
        </PageContent>
      </div>
    )
  }
}

const mapStateToProps = (state) => ({
  core: state.core,
  onTestnet: state.core.onTestnet,
  testnetBlockchainMismatch: testnetUtils.isBlockchainMismatch(state),
  testnetNetworkMismatch: testnetUtils.isNetworkMismatch(state),
  testnetNextReset: state.testnet.nextReset,
})

const mapDispatchToProps = () => ({})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Index)
