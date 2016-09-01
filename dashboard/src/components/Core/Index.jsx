import chain from '../../chain'
import { context } from '../../utility/environment'

import React from 'react'
import PageHeader from '../PageHeader/PageHeader'
import ErrorBanner from '../Common/ErrorBanner'
import Panel from '../Common/Panel'

import styles from "./Index.scss"

export default class Index extends React.Component {
  constructor(props) {
    super(props)
    this.state = {}
    this.deleteClick = this.deleteClick.bind(this)
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

    return (
      <div>

        <PageHeader additionalStyles={styles.page_header} title='Core'/>

        <div className={`${styles.top} ${styles.flex}`}>
          <div className={`${styles.left} ${styles.col}`}>
            <div>
              <h3>Configuration</h3>
              <table className={styles.table}>
                <tbody>
                  <tr>
                    <td className={styles.row_label}>Core Type:</td>
                    <td>{this.props.core.generator ? 'Generator' : 'Node'}</td>
                  </tr>
                  <tr>
                    <td className={styles.row_label}>Setup Time:</td>
                    <td>{this.props.core.configuredAt}</td>
                  </tr>
                  <tr>
                    <td className={styles.row_label}>Build Commit:</td>
                    <td>{this.props.core.buildCommit}</td>
                  </tr>

                  {this.props.core.generator && <tr>
                    <td className={styles.row_label}>Initial Block hash:</td>
                    <td><code className={styles.block_hash}>{this.props.core.initialBlockHash}</code></td>
                  </tr>}

                </tbody>
              </table>
            </div>
          </div>

          <div className={`${styles.col}`}>
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

              {!this.props.core.generator &&
                <p>
                  You may be experiencing latency in receiving funds and issuing
                  assets due to the replication lag in the network. Check out the
                  Core documentation on how to fix this issue.
                </p>}
            </div>
          </div>
        </div>

        <div className="row">
          <div className="col-sm-6">
            <h3>Reset Data</h3>
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
          </div>
        </div>
      </div>
    )
  }

  deleteClick() {
    if (!window.confirm("Are you sure you want to delete all data on this core?")) {
      return
    }

    this.setState({deleteDisabled: true})

    chain.Core.reset(context).then(() => {
      // TODO: Use Redux state reset and nav action instead of window.location.
      // Also, move confirmation message to a bonafide flash div. alert() in a
      // browser microtask is going away. cf https://www.chromestatus.com/features/5647113010544640
      window.alert("Data on this core has been reset. The dashboard will now reload")
      window.location.href = '/'
    }).catch((err) => {
      this.setState({
        deleteError: err,
        deleteDisabled: false,
      })
    })
  }
}
