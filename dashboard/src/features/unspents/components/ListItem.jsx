import React from 'react'
import { KeyValueTable, RawJsonButton, } from 'features/shared/components'
import { buildInOutDisplay } from 'features/transactions/utility'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}
    delete item.id
    return(<KeyValueTable
            title={
              <span>
                Transaction <code>{item.transaction_id}</code> -
                Position <code>{item.position}</code>
              </span>
             }
            actions={[
              <RawJsonButton key='raw-json' item={item} />
            ]}
            items={buildInOutDisplay(item)} />)
  }
}

export default ListItem
