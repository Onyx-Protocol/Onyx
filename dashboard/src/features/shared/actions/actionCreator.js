export default function(type, payloadCreator) {
  let creator
  if (payloadCreator) {
    creator = (...args) => {
      return {...payloadCreator(...args), type}
    }
  } else {
    creator = () => {
      return {type}
    }
  }

  creator.type = type

  return creator
}
