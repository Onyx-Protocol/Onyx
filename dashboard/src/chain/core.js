export default class Core {
  static reset(context) {
    return context.client.request('/reset')
  }

  static configure(body, context) {
    return context.client.request('/configure', body)
  }

  static info(context) {
    return context.client.request('/info')
  }
}
