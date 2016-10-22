import React from 'react'
import moment from 'moment'
import { humanizeDuration } from 'utility/time'

class RelativeTime extends React.Component {
  render() {
    let timestamp = moment(this.props.timestamp).fromNow()

    const diff = moment(this.props.timestamp).diff(moment())
    if (diff > 0) {
      timestamp = humanizeDuration(diff/1000) + ' ahead of local time'
    }

    return(
      <span title={this.props.timestamp}>{timestamp}</span>
    )
  }
}

export default RelativeTime
