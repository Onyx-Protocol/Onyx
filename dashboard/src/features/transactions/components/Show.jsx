import React from 'react'
import { actions } from '../'
import { connect } from 'react-redux'

class Show extends React.Component {
  componentDidMount() {
    this.props.fetchItem(this.props.params.id)
  }

  render() {
    const item = this.props.item

    if (item) {
      return(
        <div className='panel panel-default'>
          <div className='panel-heading'>
            <strong>Transaction - {item.id}</strong>
          </div>
          <div className='panel-body'>
            <pre>
              {JSON.stringify(item, null, '  ')}
            </pre>
          </div>
        </div>
      )
    } else {
      return(<div>Loading...</div>)
    }
  }
}

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
