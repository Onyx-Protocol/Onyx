import React from 'react'
import {
  BaseShow,
  PageContent,
  PageTitle,
  KeyValueTable,
  RawJsonButton,
} from 'features/shared/components'
import componentClassNames from 'utility/componentClassNames'

class AssetShow extends BaseShow {
  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Asset '}
        <code>{item.alias ? item.alias :item.id}</code>
      </span>

      view = <div className={componentClassNames(this)}>
        <PageTitle title={title} />

        <PageContent>
          <KeyValueTable
            id={item.id}
            object='asset'
            title='Details'
            actions={[
              <button key='show-circulation' className='btn btn-link' onClick={this.props.showCirculation.bind(this, item)}>
                Circulation
              </button>,
              <RawJsonButton key='raw-json' item={item} />
            ]}
            items={[
              {label: 'ID', value: item.id},
              {label: 'Alias', value: item.alias},
              {label: 'Tags', value: item.tags, editUrl: `/assets/${item.id}/tags`},
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
                {label: 'Root Xpub', value: key.rootXpub},
                {label: 'Asset Pubkey', value: key.assetPubkey},
                {label: 'Asset Derivation Path', value: key.assetDerivationPath},
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
)(AssetShow)
