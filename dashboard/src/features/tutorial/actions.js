const dismissTutorial = { type: 'DISMISS_TUTORIAL' }
const openTutorial = { type: 'OPEN_TUTORIAL' }
const tutorialNextStep =  { type: 'TUTORIAL_NEXT_STEP' }
const changeTutorialRoute = { type: 'TUTORIAL_PAGE_ROUTE' }

let actions = {
  dismissTutorial,
  openTutorial,
  tutorialNextStep,
  changeTutorialRoute
}

export default actions
