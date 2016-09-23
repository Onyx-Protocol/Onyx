import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Transaction/Item'

const type = 'transaction'

export default connect(
  mapStateToProps(type, Item),
  mapDispatchToProps(type)
)
