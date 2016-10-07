import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'account'

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem),
  BaseList.mapDispatchToProps(type)
)
