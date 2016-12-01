import React from 'react'

class RawJsonButton extends React.Component {
  showRawJson(item){
    this.props.showRawJson(<pre>{JSON.stringify(item, null, 2)}</pre>)
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
  showRawJson: (body) => dispatch(actions.app.showModal(
    body,
    actions.app.hideModal(),
    null,
    { wide: true }
  )),
})

export default connect(
  () => ({}),
  mapDispatchToProps
)(RawJsonButton)
