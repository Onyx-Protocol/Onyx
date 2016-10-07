import { baseListActions, baseFormActions } from 'features/shared/actions'

const type = 'mockhsm'

export default {
  ...baseFormActions(type, { className: 'MockHsm' }),
  ...baseListActions(type, { className: 'MockHsm' }),
}
