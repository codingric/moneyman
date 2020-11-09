import React from "react";

class Account extends React.Component {
  render() {
    return (
    <li>{this.props.name} ({this.props.number})</li>
    )
  }
}

export default Account;
