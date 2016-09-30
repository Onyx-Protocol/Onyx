import React from 'react'
import { BaseShow } from '../../shared/components'

class Show extends BaseShow {
  render() {
    const item = this.props.item

    let view
    if (item) {
      view = <div className='panel panel-default'>
        <div className='panel-heading'>
          <strong>Transaction - {item.id}</strong>
        </div>
        <div className='panel-body'>
          <pre>
            {JSON.stringify(item, null, '  ')}
          </pre>
        </div>
      </div>

    }
    return this.renderIfFound(view)
  }
}

// Container

import { actions } from '../'
import { connect } from 'react-redux'

const mapStateToProps = (state, ownProps) => ({
  item: state.transaction.items[ownProps.params.id]
})

const mapDispatchToProps = ( dispatch ) => ({
  fetchItem: (id) => dispatch(actions.fetchItems({filter: `id='${id}'`}))
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Show)
