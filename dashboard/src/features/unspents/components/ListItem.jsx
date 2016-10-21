import React from 'react'
import { KeyValueTable } from 'features/shared/components'
import { buildInOutDisplay } from 'features/transactions/utility'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}
    delete item.id

    return(<KeyValueTable items={buildInOutDisplay(item)} />)
  }
}

export default ListItem
