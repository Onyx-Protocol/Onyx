import React from 'react'
import { CopyableBlock } from 'features/shared/components'

export default class CreateModal extends React.Component {
  render() {
    return <div>
      <h4>Created new access token</h4>
      <p>Please store this token carefully. This is the last time it will be displayed.</p>

      <CopyableBlock value={this.props.token} />
    </div>
  }
}
