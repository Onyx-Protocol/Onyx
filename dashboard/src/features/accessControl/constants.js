export const policyOptions = [
  'client-readwrite',
  'client-readonly',
  'network',
  'monitoring',
].map(val => ({label: val, value: val}))
