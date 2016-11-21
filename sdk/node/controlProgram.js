export default class ControlProgram {
  static create(body, context) {
    return context.client.request('/create-control-program', body)
      .then(data => data[0])
  }
}
