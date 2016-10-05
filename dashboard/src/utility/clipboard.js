export const copyToClipboard = value => {
  const listener = e => {
    e.clipboardData.setData('text/plain', value)
    e.preventDefault()
    document.removeEventListener('copy', listener)
  }
  document.addEventListener('copy', listener)
  document.execCommand('copy')
}
