import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../features/accounts/components/ListItem'

const type = 'account'

export default connect(
  mapStateToProps(type, Item),
  mapDispatchToProps(type)
)
