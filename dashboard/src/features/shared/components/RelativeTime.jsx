import React from 'react'
import moment from 'moment'

class RelativeTime extends React.Component {
  render() {
    const timestamp = moment(this.props.timestamp).fromNow()
    return(
      <span title={this.props.timestamp}>{timestamp}</span>
    )
  }
}

export default RelativeTime
