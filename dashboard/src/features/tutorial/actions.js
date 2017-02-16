function tutorialNextStep(route){
  return { type: 'TUTORIAL_NEXT_STEP', route }
}
function submitTutorialForm(data, object){
  return { type: 'UPDATE_TUTORIAL', object, data }
}

let actions = {
  tutorialNextStep,
  submitTutorialForm
}

export default actions
