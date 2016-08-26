export default class Core {
  static reset(context) {
    return context.client.request('/reset')
  }
}
