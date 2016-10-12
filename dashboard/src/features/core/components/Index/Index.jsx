import { connect } from 'react-redux'
import { context } from 'utility/environment'
import { getUpcomingReset, getNetworkMismatch } from 'features/configuration/reducers'
import chain from 'chain'
import { ErrorBanner } from 'features/shared/components'
import PageHeader from 'components/PageHeader/PageHeader'
import React from 'react'
import styles from './Index.scss'
import moment from 'moment-timezone'

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
    let replicationLagClass
    if (this.props.core.replicationLag < 5) {
      replicationLagClass = styles.green
    } else if (this.props.core.replicationLag < 10) {
      replicationLagClass = styles.yellow
    } else {
      replicationLagClass = styles.red
    }

    let configBlock = (
      <div className={`${styles.left} ${styles.col}`}>
        <div>
          <h3>Configuration</h3>
          <table className={styles.table}>
            <tbody>
              <tr>
                <td className={styles.row_label}>Core Type:</td>
                <td>{this.props.core.coreType}</td>
              </tr>
              <tr>
                <td className={styles.row_label}>Setup Time:</td>
                <td>{this.props.core.configuredAt}</td>
              </tr>
              <tr>
                <td className={styles.row_label}>Build:</td>
                <td><code>{this.props.core.buildCommit}</code></td>
              </tr>
              <tr>
                <td colSpan={2}><hr /></td>
              </tr>
              <tr>
                <td className={styles.row_label}>Generator URL:</td>
                <td>{this.props.core.generator ? window.location.origin : this.props.core.generatorUrl}</td>
              </tr>
              {!this.props.core.generator &&
                <tr>
                  <td className={styles.row_label}>Generator Access Token:</td>
                  <td><code>{this.props.core.generatorAccessToken}</code></td>
                </tr>}
              <tr>
                <td className={styles.row_label}>Blockchain ID:</td>
                <td><code className={styles.block_hash}>{this.props.core.blockchainID}</code></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    )

    let networkStatusBlock = (
      <div className={`${styles.right} ${styles.col}`}>
        <div>
          <h3>Network Status</h3>

          <table className={styles.table}>
            <tbody>
              <tr>
                <td className={styles.row_label}>Generator Block:</td>
                <td>{this.props.core.generatorBlockHeight}</td>
              </tr>
              <tr>
                <td className={styles.row_label}>Local Block:</td>
                <td>{this.props.core.blockHeight}</td>
              </tr>

              {this.props.core.replicationLag && <tr>
                <td className={styles.row_label}>Replication Lag:</td>
                <td className={`${styles.replication_lag} ${replicationLagClass}`}>
                  {this.props.core.replicationLag}
                </td>
              </tr>}
            </tbody>
          </table>

          {this.props.onTestNet &&
            <div>
              {this.props.networkMismatchError && <ErrorBanner
                title='Network RPC Version'
                message='This core is not compatible with the Testnet generator. Please update to the latest official release of Chain Core.'
                />}

              {this.props.upcomingResetError && <ErrorBanner
                title='Network Reset'
                message={`The Testnet will be reset at ${this.props.resetTime}`}
                />}
            </div>}
        </div>
      </div>
    )

    let resetDataBlock = (
      <div className='row'>
        <div className='col-sm-6'>
          <h3>Reset Data</h3>

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
                className='btn btn-danger btn-lg'
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
      <div>
        <PageHeader additionalStyles={styles.page_header} title='Core'/>

        <div className={`${styles.top} ${styles.flex}`}>
          {configBlock}
          {!this.props.core.generator && networkStatusBlock}
        </div>

        {resetDataBlock}
      </div>
    )
  }
}

// Container

const mapStateToProps = (state) => ({
  core: state.core,
  onTestNet: state.core.onTestNet,
  upcomingResetError: getUpcomingReset(state),
  resetTime: moment(state.configuration.testNetResetTime).tz('UTC').format('MMMM Do YYYY, h:mm:ss z'),
  networkMismatchError: getNetworkMismatch(state),
})

const mapDispatchToProps = () => ({})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Index)
