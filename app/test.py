import flask
import flask_sqlalchemy
import flask_restless
from models import *
from sqlalchemy import Column, Float, Integer, String, Date, Enum
import enum
import os

# Create the Flask application and the Flask-SQLAlchemy object.
app = flask.Flask(__name__,static_folder="../static/dist")
app.config['DEBUG'] = True
app.config['SQLALCHEMY_DATABASE_URI'] = os.environ.get('DATABASE','sqlite:///moneyman.sqlite')
db = flask_sqlalchemy.SQLAlchemy(app)

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

class Payment(db.Model):
    __tablename__ = 'payment'
    id = Column(Integer, primary_key=True, autoincrement=True)
    plan_id = db.Column(db.Integer, db.ForeignKey('plan.id'))
    plan = db.relationship('Plan', backref=db.backref('payments', lazy='dynamic'))
    description = Column(String(1024))
    account = Column(Integer)
    amount = Column(Float())
    tag = Column(String(128))
    date = Column(Date)

class Plan(db.Model):
    __tablename__ = 'plan'
    id = Column(Integer, primary_key=True, autoincrement=True)
    name = Column(String(64))
    description = Column(String(1024))
    amount = Column(Float())
    frequency = Column(String(32))
    type = Column(String(32))
    tag = Column(String(128))
    account_from = Column(Integer())
    account_to = Column(Integer())
    regex = Column(String(1024))

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
    target.ref = hashlib.md5("%s %s %s".format(target.date, target.account, target.description).encode('utf-8')).hexdigest()

# Create the database tables.
db.create_all()

# Create the Flask-Restless API manager.
manager = flask_restless.APIManager(app, flask_sqlalchemy_db=db)

# Create API endpoints, which will be available at /api/<tablename> by
# default. Allowed HTTP methods can be specified as well.
manager.create_api(Transfer, methods=['GET', 'POST'])
manager.create_api(Payment, methods=['GET', 'POST', 'PUT', 'DELETE'])
manager.create_api(Plan, methods=['GET', 'POST'])
manager.create_api(Account, methods=['GET', 'POST'])


# manager.create_api(Computer, methods=['GET'])

# start the flask loop

@app.route('/')
def react():
    return flask.render_template("react.html")

app.run()