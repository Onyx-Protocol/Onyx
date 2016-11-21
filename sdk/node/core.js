export default class Core {
  static reset(context) {
    return context.client.request('/reset', {everything: true})
  }

  static configure(context, body) {
    return context.client.request('/configure', body)
  }

  static updateConfiguration(context, body) {
    return context.client.request('/update-configuration', body)
  }

  static info(context) {
    return context.client.request('/info')
  }
}
