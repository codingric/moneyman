from sqlalchemy import create_engine
from sqlalchemy.orm import scoped_session, sessionmaker
import os
from sqlalchemy.ext.declarative import declarative_base
engine = create_engine('mysql+pymysql://market:show-me-the-money@market.cl87rnyncckc.ap-southeast-2.rds.amazonaws.com/moneyman', encoding='utf-8')
db_session = scoped_session(sessionmaker(autocommit=False,
                                         autoflush=False,
                                         bind=engine))

db_session.text_factory = str

Base = declarative_base()
Base.query = db_session.query_property()

if __name__ == '__main__':
    # import all modules here that might define models so that
    # they will be registered properly on the metadata.  Otherwise
    # you will have to import them first before calling init_db()
    from models import *
    Base.metadata.create_all(bind=engine)
