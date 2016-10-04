import { mapStateToProps, mapDispatchToProps, connect } from 'containers/Base/List'
import { ListItem } from 'features/transactions/components'

const type = 'transaction'

export default connect(
  mapStateToProps(type, ListItem),
  mapDispatchToProps(type)
)
