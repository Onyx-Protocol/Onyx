const dismissTutorial = { type: 'DISMISS_TUTORIAL' }
const openTutorial = { type: 'OPEN_TUTORIAL' }
const tutorialNextStep =  { type: 'TUTORIAL_NEXT_STEP' }
function updateTutorial(data, object){
  var info = {}
  info[object] = data
  return { type: 'UPDATE_TUTORIAL', info }
}

let actions = {
  dismissTutorial,
  openTutorial,
  tutorialNextStep,
  updateTutorial
}

export default actions
