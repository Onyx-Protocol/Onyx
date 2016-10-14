import React from 'react'
import { connect } from 'react-redux'
import { NotFound, PageTitle } from 'features/shared/components'
import styles from './GeneratedTxHex.scss'
import { copyToClipboard } from 'utility/clipboard'

class Generated extends React.Component {
  render() {
    if (!this.props.hex) return <NotFound />

    return (
      <div>
        <PageTitle title='Generated Transaction' />

        <div className={styles.main}>
          <p>Use the following hex string as the base transaction for a future transaction:</p>

          <button
            className='btn btn-primary'
            onClick={() => copyToClipboard(this.props.hex)}
          >
            Copy to clipboard
          </button>

          <pre className={styles.hex}>{this.props.hex}</pre>
        </div>
      </div>
    )
  }
}

export default connect(
  // mapStateToProps
  (state, ownProps) => {
    const generated = (state.transaction || {}).generated || []
    const found = generated.find(i => i.id == ownProps.params.id)
    if (found) return {hex: found.hex}
    return {}
  }
)(Generated)
