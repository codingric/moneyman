from sqlalchemy import Column, Float, Integer, String, Date, Enum
from database import Base
import enum

class FrequencyEnum(enum.Enum):
  daily = 365.25
  weekly = 365.25/7
  fortnightly = 365.25/14
  monthly = 12
  quarterly = 4
  biannually = 2
  annually = 1

class Transaction(Base):
  __tablename__ = 'transactions'
  id = Column(Integer, primary_key=True, autoincrement=True)
  ref = Column(String(32), unique=True)
  description = Column(String(1024))
  account = Column(Integer)
  amount = Column(Float())
  tag = Column(String(128))
  date = Column(Date)

  def __init__(self, **kwargs):
    for k, v in kwargs.iteritems():
      setattr(self, k, v)

class Tag(Base):
  __tablename__ = 'tags'
  id = Column(Integer, primary_key=True, autoincrement=True)
  name = Column(String(64))
  description = Column(String(1024))
  budget_amount = Column(Float())
  budget_frequency = Column(Enum(FrequencyEnum))
  tag = Column(String(128))

class Matcher(Base):
  __tablename__ = 'matchers'
  id = Column(Integer, primary_key=True, autoincrement=True)
  name = Column(String(128))
  regex = Column(String(1024))
  tag = Column(String(128))
  account = Column(Integer())

class Account(Base):
  __tablename__ = 'accounts'
  number = Column(Integer, primary_key=True)
  name = Column(String(128))
  balance = Column(Float())
  balance_date = Column(Date)
