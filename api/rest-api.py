import flask
import flask_sqlalchemy
import flask_restless
from sqlalchemy import Column, Float, Integer, String, Date, Enum, Unicode
import enum
import os
from flask_jwt import JWT, jwt_required, current_identity
from flask_cors import CORS

DATABASE = os.environ.get('DATABASE','sqlite:////tmp/moneyman.sqlite')

# Create the Flask application and the Flask-SQLAlchemy object.
app = flask.Flask(__name__,static_folder="./dist")
app.config['DEBUG'] = True
app.config['SQLALCHEMY_DATABASE_URI'] = DATABASE
app.config["JWT_SECRET_KEY"] = "6174AF5FAD1CAF4E7558DB85343FEC509CDC8C719FD6D9DD57329EF9A7D1BB51"
db = flask_sqlalchemy.SQLAlchemy(app)
CORS(app, resources={r"/api/*": {"origins": "*"}})

class User(object):

    def __init__(self, id):
        self.id = str(id)

def authenticate(username, password):
    return User(1)

def identity(payload):
    return User(1)

@jwt_required()
def auth_func(**kw):
    pass

jwt = JWT(app, authenticate, identity)


# Create your Flask-SQLALchemy models as usual but with the following two
# (reasonable) restrictions:
#   1. They must have a primary key column of type sqlalchemy.Integer or
#      type sqlalchemy.Unicode.
#   2. They must have an __init__ method which accepts keyword arguments for
#      all columns (the constructor in flask_sqlalchemy.SQLAlchemy.Model
#      supplies such a method, so you don't need to declare a new one).
# class Person(db.Model):
#     id = db.Column(db.Integer, primary_key=True)
#     name = db.Column(db.Unicode, unique=True)
#     birth_date = db.Column(db.Date)


# class Computer(db.Model):
#     id = db.Column(db.Integer, primary_key=True)
#     name = db.Column(db.Unicode, unique=True)
#     vendor = db.Column(db.Unicode)
#     purchase_time = db.Column(db.DateTime)
#     owner_id = db.Column(db.Integer, db.ForeignKey('person.id'))
#     owner = db.relationship('Person', backref=db.backref('computers',
#                                                          lazy='dynamic'))

class Transfer(db.Model):
    __tablename__ = 'transfer'
    id = Column(Integer, primary_key=True, autoincrement=True)
    ref = Column(String(32), unique=True)
    description = Column(String(1024))
    account = Column(Integer)
    amount = Column(Float())
    tag = Column(String(128))
    date = Column(Date)

class Due(db.Model):
    __tablename__ = 'due'
    id = Column(Integer, primary_key=True, autoincrement=True)
    description = Column(String(1024), nullable=False)
    account = Column(Integer)
    amount = Column(Float())
    tag = Column(String(128))
    date = Column(Date)

class Recurring(db.Model):
    __tablename__ = 'recurring'
    id = Column(Integer, primary_key=True, autoincrement=True)
    description = Column(String(1024), nullable=False)
    account_from = Column(Integer())
    account_to = Column(Integer())
    tag = Column(String(128))
    rrule = Column(String(1024),nullable=False)
    matcher = Column(String(1024))

class Budget(db.Model):
    __tablename__ = 'budget'
    id = Column(Integer, primary_key=True, autoincrement=True)
    name = Column(String(64))
    description = Column(String(1024))
    amount = Column(Float())
    frequency = Column(String(32))
    tag = Column(String(128))
    account_from = Column(Integer())
    account_to = Column(Integer())
    start_date = Column(String(32))
    end_date = Column(String(32))
    amount_moved = Column(Float())

class Account(db.Model):
    __tablename__ = 'account'
    number = Column(Integer, primary_key=True)
    name = Column(String(128))
    balance = Column(Float())
    balance_date = Column(Date)

from sqlalchemy import event
import hashlib
@event.listens_for(Transfer, 'before_insert')
def receive_before_insert(mapper, connection, target):
    ref = f"{target.date} {target.account} {target.description}".encode('utf-8')
    target.ref = hashlib.md5(ref).hexdigest()


# Create the database tables.
db.create_all()

# Create the Flask-Restless API manager.
manager = flask_restless.APIManager(app, flask_sqlalchemy_db=db)

jwt_required=dict(GET_SINGLE=[auth_func], GET_MANY=[auth_func])

# Create API endpoints, which will be available at /api/<tablename> by
# default. Allowed HTTP methods can be specified as well.
manager.create_api(Transfer, methods=['GET', 'POST'])
manager.create_api(Due, methods=['GET'])
manager.create_api(Recurring, methods=['GET', 'POST', 'PUT', 'DELETE'])
manager.create_api(Budget, methods=['GET', 'POST', 'PUT', 'DELETE'])
manager.create_api(Account, methods=['GET', 'POST'])

# manager.create_api(Computer, methods=['GET'])

# start the flask loop

@app.route('/')
def react():
    return flask.render_template("react.html")

@app.route('/delete')
def remove():
    os.remove(DATABASE[10:])
    return "Deleted"

@app.route('/create')
def create():
    db.create_all()
    return "Created"

@app.route('/globals')
def show_globals():
    return {"DATABASE": DATABASE}

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=os.environ.get("PORT", 5000))