const dismissTutorial = { type: 'DISMISS_TUTORIAL' }
const openTutorial = { type: 'OPEN_TUTORIAL' }
function tutorialNextStep(route){
  return { type: 'TUTORIAL_NEXT_STEP', route }
}
function updateTutorial(data, object){
  return { type: 'UPDATE_TUTORIAL', object, data }
}

let actions = {
  dismissTutorial,
  openTutorial,
  tutorialNextStep,
  updateTutorial
}

export default actions
