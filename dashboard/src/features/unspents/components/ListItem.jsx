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
                ID <code>{item.id}</code>
              </span>
             }
            actions={[
              <RawJsonButton key='raw-json' item={item} />
            ]}
            items={buildInOutDisplay(item)} />)
  }
}

export default ListItem
