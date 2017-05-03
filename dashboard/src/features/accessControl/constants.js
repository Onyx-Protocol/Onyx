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
    label: 'Monitoring',
    value: 'monitoring',
    hint: 'Access to monitoring-specific endpoints'
  },
  {
    label: 'Cross-core',
    value: 'crosscore',
    hint: 'Access to the cross-core API, not including block-signing. Necessary for connecting to the generator'
  },
  {
    label: 'Cross-core block signing',
    value: 'crosscore-signblock',
    hint: 'Access to the cross-core API\'s block-signing functionality'
  },
  {
    label: 'Internal',
    value: 'internal',
    hidden: true,
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
