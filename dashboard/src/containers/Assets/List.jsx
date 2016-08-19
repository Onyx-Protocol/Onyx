import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Asset/Item'

const type = "asset"

export default connect(
  mapStateToProps(type, Item),
  mapDispatchToProps(type)
)
