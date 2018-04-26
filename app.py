# vim: set ts=2 sw=2 sts=2

from moneyman import Importer, INGDIRECT_CONFIG
from flask import Flask, request
from database import db_session
from models import *
import json
import sqlalchemy
from sqlalchemy import func, and_

app = Flask(__name__)

@app.route('/')
def index():
    return 'Index'

@app.route('/upload', methods=['GET','POST'])
def upload():
  if request.method == 'POST':
    if 'csv' not in request.files or request.files['csv'].filename == '':
      return "No file supplied", 403

    file = request.files['csv']
    file.save("./import.csv")

    last = db_session.query(Transaction.ref, Transaction.id, func.max(Transaction.date)).first()

    ref, id, last_date = last

    i = Importer(INGDIRECT_CONFIG, file_path="./import.csv")
    imported = 0
    for row in i.read():
      if row['ref']:
        break
      try:
        transaction = Transaction(**row)
        db_session.add(transaction)
        db_session.commit()
        imported += 1
      except:
        break

    return "Imported %d" % imported, 200
  return '''
    <!doctype html>
    <title>Upload new File</title>
    <h1>Upload new File</h1>
    <form method=post enctype=multipart/form-data>
      <p><input type=file name=csv>
         <input type=submit value=Upload>
    </form>
    '''

@app.teardown_appcontext
def shutdown_session(exception=None):
    db_session.remove()

if __name__ == '__main__':
  app.run(debug=True,host='0.0.0.0',port=8000)
