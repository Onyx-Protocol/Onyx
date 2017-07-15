import React from 'react'
import { CopyableBlock } from 'features/shared/components'
import componentClassNames from 'utility/componentClassNames'

export default class ReceiverModal extends React.Component {
  render() {
    return <div className={componentClassNames(this)}>
      <p>Copy this one-time use receiver to use in a transaction:</p>
      <CopyableBlock value={JSON.stringify(this.props.receiver, null, 1)} />
    </div>
  }
}
