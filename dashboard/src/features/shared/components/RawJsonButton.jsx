import React from 'react'
import { Connection } from 'chain-sdk'

class RawJsonButton extends React.Component {
  showRawJson(item){
    const snakeCased = Connection.snakeize(item)
    this.props.showJsonModal(<pre>{JSON.stringify(snakeCased, null, 2)}</pre>)
  }

  render() {
    return (
        <button className='btn btn-link' onClick={this.showRawJson.bind(this, this.props.item)}>
          Raw JSON
        </button>
    )
  }
}

import { connect } from 'react-redux'
import actions from 'actions'

const mapDispatchToProps = ( dispatch ) => ({
  showJsonModal: (body) => dispatch(actions.app.showModal(
    body,
    actions.app.hideModal,
    null,
    { wide: true }
  )),
})

export default connect(
  () => ({}),
  mapDispatchToProps
)(RawJsonButton)
