// Add irregular plurals
var _pluralize = require('pluralize')
_pluralize.addIrregularRule('index', 'indexes')

export const pluralize = _pluralize

export const capitalize = (string) => {
  return string.charAt(0).toUpperCase() + string.slice(1)
}
