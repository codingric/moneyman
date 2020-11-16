
import React from "react";
import Accounts from './Accounts';
import Budgets from './Budgets';
import Budget from './Budget';
import NewBudget from './NewBudget';

import { BrowserRouter as Router, Route, Switch, Link } from "react-router-dom"

import { Menu } from 'semantic-ui-react'


export default class App extends React.Component {
    state = {}

    handleItemClick = (e, { name }) => this.setState({ activeItem: name })
  
    render() {
      const { activeItem } = this.state
        return (
            <div className="App">
                <Router>
                    <Menu>
                        <Menu.Item link={true} name='home' onClick={this.handleItemClick} active={activeItem === 'home'}><Link to="/">Home</Link></Menu.Item>
                        <Menu.Item name='accounts' onClick={this.handleItemClick} active={activeItem === 'accounts'}><Link to="/accounts">Accounts</Link></Menu.Item>
                        <Menu.Item name='budgets' onClick={this.handleItemClick} active={activeItem === 'budgets'}><Link to="/budgets">Budgets</Link></Menu.Item>
                    </Menu>
                    <Switch>
                        <Route path="/accounts" exact component={()=>(<Accounts />)} />
                        <Route path="/budgets" exact component={() => (<Budgets />)} />
                        <Route path="/budget/:id" exact component={ Budget } />
                        <Route path="/budgets/new" exact component={ NewBudget } />
                    </Switch>
                </Router>

            </div>
        )
    }
}