const dismissTutorial = { type: 'DISMISS_TUTORIAL' }
const openTutorial = { type: 'OPEN_TUTORIAL' }
const tutorialNextStep =  { type: 'TUTORIAL_NEXT_STEP' }
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
