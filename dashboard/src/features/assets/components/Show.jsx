import React from 'react'
import {
  BaseShow,
  PageContent,
  PageTitle,
  KeyValueTable,
  RawJsonButton,
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

        <PageContent>
          <KeyValueTable
            title='Details'
            actions={[
              <button className='btn btn-link' onClick={this.props.showCirculation.bind(this, item)}>
                Circulation
              </button>,
              <RawJsonButton key='raw-json' item={item} />
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
            <KeyValueTable
              key={index}
              title={`Key ${index + 1}`}
              items={[
                {label: 'Index', value: index},
                {label: 'Root Xpub', value: key.root_xpub},
                {label: 'Asset Pubkey', value: key.asset_pubkey},
                {label: 'Asset Derivation Path', value: key.asset_derivation_path},
              ]}
            />
          )}
        </PageContent>
      </div>
    }
    return this.renderIfFound(view)
  }
}

// Container

import { connect } from 'react-redux'
import actions from 'actions'

const mapStateToProps = (state, ownProps) => ({
  item: state.asset.items[ownProps.params.id]
})

const mapDispatchToProps = ( dispatch ) => ({
  fetchItem: (id) => dispatch(actions.asset.fetchItems({filter: `id='${id}'`})),
  showCirculation: (item) => {
    let filter = `asset_id='${item.id}'`
    if (item.alias) {
      filter = `asset_alias='${item.alias}'`
    }

    dispatch(actions.balance.pushList({ filter }))
  },
})


export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Show)
