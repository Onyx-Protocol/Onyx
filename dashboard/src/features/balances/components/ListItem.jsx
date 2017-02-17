import React from 'react'
import { KeyValueTable } from 'features/shared/components'
import { buildBalanceDisplay } from 'utility/buildInOutDisplay'

class ListItem extends React.Component {
  render() {
    return <KeyValueTable items={buildBalanceDisplay(this.props.item)} />
  }
}

export default ListItem
