import React from 'react'
import { KeyValueTable, RawJsonButton, } from 'features/shared/components'
import buildInOutDisplay from 'utility/buildInOutDisplay'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}
    const id = item.id
    delete item.id
    return(<KeyValueTable
            title={
              <span>
                ID <code>{id}</code>
              </span>
             }
            actions={[
              <RawJsonButton key='raw-json' item={item} />
            ]}
            items={buildInOutDisplay(item)} />)
  }
}

export default ListItem
