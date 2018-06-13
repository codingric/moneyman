import React from "react";
import Account from './Account'

export default class Accounts extends React.Component {

  constructor(props) {
    super(props);

    this.state = { objects: []};
  }

  componentDidMount() {
    fetch("/api/account")
      .then(json => json.json())
      .then(data => { this.setState(data) })
      .then(data => { console.log(this.state); });
  }

  render() {
    return (
    <div>
      <h2>Accounts</h2>
      {this.state.objects.map(account => <Account key={account.number} name={account.name} number={account.number} />)}
    </div>
    )
  }
}
