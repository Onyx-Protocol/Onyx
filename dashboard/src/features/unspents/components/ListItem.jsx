import React from 'react'
import { KeyValueTable, RawJsonButton, } from 'features/shared/components'
import { buildInOutDisplay } from 'features/transactions/utility'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}
    delete item.id
    return(<KeyValueTable
            title={`${item.transaction_id} - ${item.position}`}
            actions={[
              <RawJsonButton key='raw-json' item={item} title={`utxo-${item.transaction_id}-${item.position}.json`}/>
            ]}
            items={buildInOutDisplay(item)} />)
  }
}

export default ListItem
