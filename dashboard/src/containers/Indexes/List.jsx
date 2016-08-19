import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Index/Item'

const type = "index"

const newStateToProps = (state) => {
  let defaults = mapStateToProps(type, Item)(state)
  defaults.skipQuery = true
  return defaults
}

export default connect(
  newStateToProps,
  mapDispatchToProps(type)
)
