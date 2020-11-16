import React from "react";
import { Table } from 'semantic-ui-react'
import { Link } from "react-router-dom";


export default class Accounts extends React.Component {

  constructor(props) {
    super(props);

    this.state = { objects: []};
  }

  componentDidMount() {
    fetch("http://localhost:5000/api/account")
      .then(json => json.json())
      .then(data => { this.setState(data) })
      .then(data => { console.log(this.state); });
  }

  render() {
    return (
    <div>
      <h2>Accounts</h2>

      {this.state.objects.length ?

  <Table singleLine>
    <Table.Header>
      <Table.Row>
        <Table.HeaderCell>Name</Table.HeaderCell>
        <Table.HeaderCell>Account Number</Table.HeaderCell>
      </Table.Row>
    </Table.Header>

    <Table.Body>

    {this.state.objects.map(account => (
      <Table.Row key={account.number}>
        <Table.Cell>{account.name}</Table.Cell>
        <Table.Cell><Link to={"/account/"+account.number}>{account.number}</Link></Table.Cell>
      </Table.Row>))}
    </Table.Body>
  </Table>
  :
  <p>None</p>
    }

      
    </div>
    )
  }
}
