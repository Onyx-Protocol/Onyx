// Assumes the presence of an input element with ID #_copyInput
export const copyToClipboard = value => {
  const listener = e => {
    e.clipboardData.setData('text/plain', value)
    e.preventDefault()
    document.removeEventListener('copy', listener)
  }

  // Required for Safari. Contents of selection are not used.
  document.getElementById('_copyInput').select()

  document.addEventListener('copy', listener)
  document.execCommand('copy')
}
