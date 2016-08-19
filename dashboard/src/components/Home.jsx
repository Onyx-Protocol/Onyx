import React from 'react'
import PageHeader from "./PageHeader"

class Home extends React.Component {
  render() {
      return (
        <div>
          <PageHeader title="Dashboard" />

          <div className="jumbotron text-center">
            <img src="http://www.animatedgif.net/underconstruction/5consbar2_e0.gif" />
            <img src="http://www.textfiles.com/underconstruction/NaNapaValleyVineyard9035construction.gif" />
          </div>
        </div>
      )
  }
}

export default Home
