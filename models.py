from sqlalchemy import Column, Float, Integer, String, Date
from database import Base

class Transaction(Base):
  __tablename__ = 'transactions'
  id = Column(Integer, primary_key=True, autoincrement=True)
  ref = Column(String(32), unique=True)
  description = Column(String())
  account = Column(String(64))
  amount = Column(Float())
  tag = Column(String(128))
  date = Column(Date)

  def __init__(self, **kwargs):
    for k, v in kwargs.iteritems():
      setattr(self, k, v)
