import { baseListActions, baseCreateActions } from 'features/shared/actions'

const type = 'mockhsm'

export default {
  ...baseCreateActions(type, { className: 'MockHsm' }),
  ...baseListActions(type, { className: 'MockHsm' }),
}
