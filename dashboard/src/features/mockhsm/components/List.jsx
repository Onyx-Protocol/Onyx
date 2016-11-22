import React from 'react'
import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'mockhsm'

const EmptyContent = <div>
  <p>
    Cryptographic private keys are the primary authorization mechanism on a blockchain.
    They control both the issuance and transfer of assets. For development
    environments, Chain Core provides a convenient MockHSM.
  </p>
  <p>
    Learn more about how to create MockHSM keys and use them to sign transactions
    in the <a href="/docs/core/build-applications/keys" target="_blank">Keys</a> guide
    of the documentation.
  </p>
</div>

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    skipQuery: true,
    label: 'MockHSM keys',
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Alias', 'xpub']
    },
    emptyContent: EmptyContent
  }),
  BaseList.mapDispatchToProps(type)
)
