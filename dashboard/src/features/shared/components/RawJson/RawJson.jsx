import React from 'react'
import styles from './RawJson.scss'

class RawJson extends React.Component {
  render() {
    return (
      <div>
        <button className='btn btn-link' onClick={this.props.toggleJson}>
          {this.props.jsonVisible ? 'Hide' : 'Show'}
          {' '}
          JSON
        </button>

        {this.props.jsonVisible && <pre>
          {JSON.stringify(this.props.item, null, ' ')}
        </pre>}
      </div>
    )
  }
}

export default RawJson
