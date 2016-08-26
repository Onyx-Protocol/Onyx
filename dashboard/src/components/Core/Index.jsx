import chain from '../../chain'
import { context } from '../../utility/environment'

import React from 'react'
import PageHeader from '../PageHeader/PageHeader'
import Panel from '../Common/Panel'

import styles from "./Index.scss"

export default class Index extends React.Component {
  constructor(props) {
    super(props)
    this.state = {}
    this.deleteClick = this.deleteClick.bind(this)
  }

  render() {
    let title = <h4 className={styles.panel_heading}>Reset data</h4>

    return (
      <div className='form-container'>

        <PageHeader key='page-title' title='Core'/>

        <Panel title={title}>
          <p>This will permanently delete all data stored in this core, including blockchain data, accounts, assets, indexes, and MockHSM keys.</p>
          <button
            className='btn btn-danger btn-lg'
            onClick={this.deleteClick}
            disabled={this.state.deleteDisabled}
          >
            Delete all data
          </button>
        </Panel>

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
