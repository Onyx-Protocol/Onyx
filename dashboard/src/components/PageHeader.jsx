import React from 'react';

class PageHeader extends React.Component {
  render() {
    return (<h1 className="page-header">{this.props.title}</h1>)
  }
}

export default PageHeader
