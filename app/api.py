import flask_restless
import flask
from models import *
from database import db_session as s

app = flask.Flask(__name__, static_folder="../static/dist")
manager = flask_restless.APIManager(app, session=s)

@app.route('/')
def react():
    return flask.render_template("react.html")

account_api = manager.create_api(Account, methods=['GET', 'PATCH', 'POST', 'DELETE'])
trans_api = manager.create_api(Transaction, methods=['GET', 'PATCH', 'POST', 'DELETE'])
budget_api = manager.create_api(Budget, methods=['GET', 'PATCH', 'POST', 'DELETE'])
matcher_api = manager.create_api(Matcher, methods=['GET', 'PATCH', 'POST', 'DELETE'])

if __name__ == '__main__':
  app.run(debug=True, host='0.0.0.0', port=5000)
