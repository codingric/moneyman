import React from "react";

export default class Budget extends React.Component {

  constructor(props) {
    super(props);

    this.delete = this.delete.bind(this);

    this.state = {};
  }

  componentDidMount() {
    fetch(`http://localhost:5000/api/budget/${this.props.match.params.id}`)
      .then(json => json.json())
      .then(data => { this.setState(data) })
      .then(data => { console.log(this.state); });
  }

  delete() {
    fetch(`http://localhost:5000/api/budget/${this.props.match.params.id}`, {
      method: 'DELETE'
    })
      .then(this.props.history.push('/budgets'));
  }

  render() {
    return (
    <div>
      <h2>Budget</h2>
      Name: {this.state.name}
      <button onClick={this.delete}>Delete</button>
    </div>
    )
  }
}
