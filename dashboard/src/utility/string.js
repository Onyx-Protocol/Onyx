import _pluralize from 'pluralize'

export const pluralize = _pluralize

export const capitalize = (string) => {
  return string.charAt(0).toUpperCase() + string.slice(1)
}

export const humanize = (string) => {
  return string
    .replace(/_/g, ' ')
}

export const parseNonblankJSON = (json) => {
  json = json || ''
  json = json.trim()

  if (json == '') {
    return null
  }

  return JSON.parse(json)
}
