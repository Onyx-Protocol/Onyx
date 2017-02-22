import React from 'react'
import { KeyValueTable, RawJsonButton, } from 'features/shared/components'
import { buildUnspentDisplay } from 'utility/buildInOutDisplay'

class ListItem extends React.Component {
  render() {
    return(<KeyValueTable
            title={
              <span>
                ID <code>{this.props.item.id}</code>
              </span>
             }
            actions={[
              <RawJsonButton key='raw-json' item={this.props.item} />
            ]}
            items={buildUnspentDisplay(this.props.item)} />)
  }
}

export default ListItem
