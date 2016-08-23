import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Transaction/Item'

const type = "transaction"

const newStateToProps = (state) => {
  let defaults = mapStateToProps(type, Item)(state)
  defaults.skipCreate = true
  return defaults
}

export default connect(
  newStateToProps,
  mapDispatchToProps(type)
)
