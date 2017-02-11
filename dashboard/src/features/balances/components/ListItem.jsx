import React from 'react'
import { KeyValueTable } from 'features/shared/components'
import buildInOutDisplay from 'utility/buildInOutDisplay'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}

    const sumBy = {}
    Object.keys(item.sumBy).forEach(key =>
      sumBy[key] = item.sumBy[key]
    )

    return(
      <KeyValueTable items={[
        {label: 'Amount', value: item.amount},
        ...buildInOutDisplay(sumBy)
      ]} />
    )
  }
}

export default ListItem
