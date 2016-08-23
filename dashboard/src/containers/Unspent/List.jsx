import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Balance/Item'

const type = "unspent"

const newStateToProps = (state) => {
  let defaults = mapStateToProps(type, Item)(state)
  defaults.skipCreate = true
  defaults.label = "Unspent Outputs"
  return defaults
}

export default connect(
  newStateToProps,
  mapDispatchToProps(type)
)
