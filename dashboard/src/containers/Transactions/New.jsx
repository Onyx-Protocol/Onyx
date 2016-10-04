import { mapStateToProps, mapDispatchToProps, connect } from 'containers/Base/New'
import { New } from 'features/transactions/components'

const type = 'transaction'

export default connect(
  mapStateToProps(type),
  mapDispatchToProps(type),
  New
)
