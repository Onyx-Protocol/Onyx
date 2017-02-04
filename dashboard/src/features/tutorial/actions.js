const dismissTutorial = { type: 'DISMISS_TUTORIAL' }
const openTutorial = { type: 'OPEN_TUTORIAL' }
function tutorialNextStep(route){
  return { type: 'TUTORIAL_NEXT_STEP', route }
}
function submitTutorialForm(data, object){
  return { type: 'UPDATE_TUTORIAL', object, data }
}

let actions = {
  dismissTutorial,
  openTutorial,
  tutorialNextStep,
  submitTutorialForm
}

export default actions
