const x509SubjectAttributes = {
  C: {array: true},
  O: {array: true},
  OU: {array: true},
  L: {array: true},
  ST: {array: true},
  STREET: {array: true},
  POSTALCODE: {array: true},
  SERIALNUMBER: {array: false},
  CN: {array: false},
}

const sanitizeX509GuardData = guardData => {
  const keys = Object.keys(guardData)
  if (keys.length !== 1 || keys[0].toLowerCase() !== 'subject') {
    throw new Error('X509 guard data must contain exactly one key, "subject"')
  }

  const newSubject = {}
  const oldSubject = guardData[keys[0]]
  for (let k in oldSubject) {
    const attrib = x509SubjectAttributes[k.toUpperCase()]
    if (!attrib) {
      throw new Error(`X509 guard data contains invalid subject attribute: ${k}`)
    }

    let v = oldSubject[k]
    if (!attrib.array && Array.isArray(v)) {
      throw new Error(`X509 guard data contains invalid array for attribute ${k}: ${v.toString()}`)
    } else if (attrib.array && !Array.isArray(v)) {
      newSubject[k] = [v]
    } else {
      newSubject[k] = v
    }
  }

  return {subject: newSubject}
}

module.exports = {
  sanitizeX509GuardData,
}
