import React from 'react'
import {
  BaseShow,
  PageTitle,
  Table,
} from 'features/shared/components'

class Show extends BaseShow {
  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Asset '}
        <code>{item.alias ? item.alias :item.id}</code>
      </span>

      view = <div>
        <PageTitle title={title} />

        <Table
          title='details'
          actions={[
            <button className='btn btn-link' onClick={this.props.showCirculation.bind(this, item)}>
              Circulation
            </button>
          ]}
          items={[
            {label: 'ID', value: item.id},
            {label: 'Alias', value: item.alias},
            {label: 'Tags', value: item.tags},
            {label: 'Definition', value: item.definition},
            {label: 'Keys', value: item.keys.length},
            {label: 'Quorum', value: item.quorum},
          ]}
        />

        {item.keys.map((key, index) =>
          <Table
            key={index}
            title={index == 0 ? 'Keys' : ''}
            items={[
              {label: 'Root Xpub', value: key.root_xpub},
              {label: 'Asset Pubkey', value: key.asset_pubkey},
              {label: 'Asset Derivation Path', value: key.asset_derivation_path},
            ]}
          />
        )}
      </div>
    }
    return this.renderIfFound(view)
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
