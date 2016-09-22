import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Balance/Item'

const type = "unspent"

const newStateToProps = (state) => ({
  ...mapStateToProps(type, Item)(state),
  skipCreate: true,
  label: 'Unspent Outputs'
})

export default connect(
  newStateToProps,
  mapDispatchToProps(type)
)
