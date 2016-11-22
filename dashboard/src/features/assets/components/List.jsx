import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'asset'

const EmptyContent = <div className="emptyContainer">
  <p>
    An asset is a type of value that can be issued on a blockchain.
    All units of a given asset are fungible. Units of an asset can be
    transacted directly between parties without the involvement of the issuer.
  </p>
  <p>
    Learn more about how to issue assets and transact between parties by
    checking out the <a href="/docs/core/build-applications/assets" target="_blank">Assets</a> guide
    in the documentation.
  </p>
</div>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Asset Alias', 'Asset ID']
    },
    emptyContent: EmptyContent
  }),
  BaseList.mapDispatchToProps(type)
)
