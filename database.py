from sqlalchemy import create_engine
from sqlalchemy.orm import scoped_session, sessionmaker
from sqlalchemy.ext.declarative import declarative_base
import os
engine = create_engine('sqlite:///moneyman.db', echo=True)
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
