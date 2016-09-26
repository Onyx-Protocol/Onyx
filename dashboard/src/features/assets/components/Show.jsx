import React from 'react'

class Show extends React.Component {
  componentDidMount() {
    this.props.fetchItem(this.props.params.id)
  }

  render() {
    const item = this.props.item

    if (item) {
      let label = item.id

      if (item.alias) {
        label = item.alias
      }

      return(
        <div className='panel panel-default'>
          <div className='panel-heading'>
            <strong>Asset - {label}</strong>
          </div>
          <div className='panel-body'>
            <pre>
              {JSON.stringify(item, null, '  ')}
            </pre>
          </div>
          <div className='panel-footer'>
            <ul className='nav nav-pills'>
              <li>
                <button className='btn btn-link' onClick={this.props.showCirculation.bind(this, item)}>
                  Circulation
                </button>
              </li>
            </ul>
          </div>

        </div>
      )
    } else {
      return(<div>Loading...</div>)
    }
  }
}

// Container

import { actions } from '../'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'
import allActions from '../../../actions'

const mapStateToProps = (state, ownProps) => ({
  item: state.asset.items[ownProps.params.id]
})

const mapDispatchToProps = ( dispatch ) => ({
  fetchItem: (id) => dispatch(actions.fetchItems({filter: `id='${id}'`})),
  showCirculation: (item) => {
    let query = `asset_id='${item.id}'`
    if (item.alias) {
      query = `asset_alias='${item.alias}'`
    }

    dispatch(allActions.balance.updateQuery(query))
    dispatch(push('/balances'))
  },
})


export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Show)
