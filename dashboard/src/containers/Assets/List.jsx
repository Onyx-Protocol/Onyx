import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import ListItem from '../../features/assets/components/ListItem'

const type = 'asset'

export default connect(
  mapStateToProps(type, ListItem),
  mapDispatchToProps(type)
)
