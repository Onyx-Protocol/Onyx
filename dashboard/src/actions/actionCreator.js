export default function(type, payloadCreator) {
  let creator
  if (payloadCreator) {
    creator = (...args) => {
      return Object.assign(payloadCreator(...args), {type})
    }
  } else {
    creator = () => {
      return {type}
    }
  }

  creator.type = type

  return creator
}
