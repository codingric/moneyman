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
  description = Column(String())
  account = Column(Integer)
  amount = Column(Float())
  tag = Column(String(128))
  date = Column(Date)

  def __init__(self, **kwargs):
    for k, v in kwargs.iteritems():
      setattr(self, k, v)

class Tag(Base):
  __tablename__ = 'tags'
  name = Column(String(64), primary_key=True)
  description = Column(String())
  budget_amount = Column(Float())
  budget_frequency = Column(Enum(FrequencyEnum))
  tag = Column(String(128))

class Matcher(Base):
  __tablename__ = 'matchers'
  id = Column(Integer, primary_key=True, autoincrement=True)
  name = Column(String())
  regex = Column(String())
  tag = Column(String(128))
  account = Column(Integer())

class Account(Base):
  __tablename__ = 'accounts'
  number = Column(Integer, primary_key=True)
  name = Column(String())
  balance = Column(Float())
  balance_date = Column(Date)
