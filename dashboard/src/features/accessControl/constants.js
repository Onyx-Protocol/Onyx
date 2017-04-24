export const policyOptions = [
  {
    label: 'Client read/write',
    value: 'client-readwrite',
    hint: 'Full access to the Client API'
  },
  {
    label: 'Client read-only',
    value: 'client-readonly',
    hint: 'Access to read-only Client endpoints'
  },
  {
    label: 'Network',
    value: 'network',
    hint: 'Access to the Network API'
  },
  {
    label: 'Monitoring',
    value: 'monitoring',
    hint: 'Access to monitoring-specific endpoints'
  },
]

export const subjectFieldOptions = [
  {label: 'CommonName', value: 'cn'},
  {label: 'Country', value: 'c', array: true},
  {label: 'Organization', value: 'o', array: true},
  {label: 'OrganizationalUnit', value: 'ou', array: true},
  {label: 'Locality', value: 'l', array: true},
  {label: 'Province', value: 'st', array: true},
  {label: 'StreetAddress', value: 'street', array: true},
  {label: 'PostalCode', value: 'postalcode', array: true},
  {label: 'SerialNumber', value: 'serialnumber'},
]
