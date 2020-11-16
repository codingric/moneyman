import React from "react";
import { Link } from "react-router-dom";
import { Button } from 'semantic-ui-react';
import 'semantic-ui-css/semantic.min.css'
import { Table, Message } from 'semantic-ui-react'
import { withRouter } from 'react-router-dom';




class Budgets extends React.Component {

  constructor(props) {
    super(props);

    this.state = { objects: []};
  }

  componentDidMount() {
    fetch("http://localhost:5000/api/budget")
      .then(json => json.json())
      .then(data => { this.setState(data) })
      .then(data => { console.log(this.state); });
  }

  render() {
    return (
    <div>
      <h2>Budgets</h2>
        <Button content="Create" primary onClick={()=>{this.props.history.push('/budgets/new')}} />


      {this.state.objects.length ?

<Table singleLine>
  <Table.Header>
    <Table.Row>
      <Table.HeaderCell>Name</Table.HeaderCell>
      <Table.HeaderCell>Frequency</Table.HeaderCell>
    </Table.Row>
  </Table.Header>

  <Table.Body>

  {this.state.objects.map(b => (
    <Table.Row key={b.id}>
      <Table.Cell><Link to={"/budget/"+b.id}>{b.name}</Link></Table.Cell>
      <Table.Cell>{b.frequency}</Table.Cell>
    </Table.Row>))}
  </Table.Body>
</Table>
:
<Message negative>
<p>No budgets found</p>
</Message>  }
    </div>
    )
  }
}

export default withRouter(Budgets)