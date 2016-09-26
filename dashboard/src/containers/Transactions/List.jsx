import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import ListItem from '../../features/transactions/components/ListItem'

const type = 'transaction'

export default connect(
  mapStateToProps(type, ListItem),
  mapDispatchToProps(type)
)
