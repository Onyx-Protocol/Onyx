import React from 'react'

class RawJsonButton extends React.Component {
  showRawJson(item){
    this.props.showRawJson(
    <div>
      <pre>
        {JSON.stringify(item, null, 2)}
      </pre>
    </div>
    )
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

const mapStateToProps = () => ({})

const mapDispatchToProps = ( dispatch ) => ({
  showRawJson: (body) => dispatch(actions.app.showModal(
    body,
    actions.app.hideModal(),
    null,
    { wide: true }
  )),
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(RawJsonButton)
