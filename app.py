# vim: set ts=2 sw=2 sts=2

from moneyman import Importer, INGDIRECT_CONFIG
from flask import Flask, request, render_template, redirect, flash, url_for, abort, session
from database import db_session
from models import *
import json
import sqlalchemy
from sqlalchemy import func, and_, text, or_, not_
from sqlalchemy.ext.serializer import loads, dumps
from datetime import date, timedelta
import uuid
import re

app = Flask(__name__)
app.secret_key = "lkaskdvmiiv887773n3nmnvfdv"

@app.before_request
def csrf_protect():
  if request.method == "POST":
    token = session.pop('_csrf_token', None)
    if not token or token != request.form.get('_csrf_token'):
      abort(403)

def generate_csrf_token():
  if '_csrf_token' not in session:
    session['_csrf_token'] = str(uuid.uuid4())
  return session['_csrf_token']

app.jinja_env.globals['csrf_token'] = generate_csrf_token

def query2json(row):
  d = []
  for o in row:
    d.append({c.name: field2json(getattr(o, c.name)) for c in o.__table__.columns})
  return d

def field2json(v):
  if isinstance(v, unicode):
    return v.encode("utf-8", "ignore")
  if typeof(v) in [float, int]:
    return v
  if v == None:
    return None
  return str(v)

@app.route('/')
def index():
    sql = text("""
      SELECT 
        count(id), 
        sum(case when amount > 0 then amount end),
        sum(case when amount < 0 then amount end)
      FROM transactions 
      WHERE date > datetime('now', '-28 days')""")
    c, p, n = db_session.execute(sql).first()
    return render_template("index.html", count=c, positive=p, negative=n)



@app.route('/tags')
def tags():
  return render_template('tag.html', tags=Tag.query.order_by(Tag.name.asc()).all())

@app.route('/transactions')
def transactions():
  return render_template('transactions.html', transactions=Transaction.query.filter(Transaction.date >= date.today() - timedelta(days=28)).filter(not_(Transaction.tag.in_(['Exclude']))).order_by(Transaction.date.desc()).all())

@app.route('/api/transactions')
def api_transactions():
  return json.dumps(query2json(Transaction.query.all()))

@app.route('/upload', methods=['GET','POST'])
def upload():
  if request.method == 'POST':
    if 'csv' not in request.files or request.files['csv'].filename == '':
      return "No file supplied", 403

    file = request.files['csv']
    file.save("./import.csv")

    ref, id, last_date = db_session.query(Transaction.ref, Transaction.id, func.max(Transaction.date)).first()

    matchers = [ (re.compile(m[0].encode('utf-8')), m[1]) for m in db_session.query(Matcher.regex, Matcher.tag).all() ]

    i = Importer(INGDIRECT_CONFIG, file_path="./import.csv")
    imported = 0
    for row in i.read():
      if row['ref'] == ref:
        break
      transaction = Transaction(**row)
      transaction.tag = find_match(transaction.description.encode('utf-8'), matchers)
      db_session.add(transaction)
      imported += 1

    db_session.commit()
    flash("Imported %d new transactions" % imported)
    return redirect(url_for('transactions'))
  return render_template('upload.html')

def find_match(t, m):
  for r, v in m:
    if r.match(t):
      print('match %s %s' % (t, v))
      return v
  return None

@app.teardown_appcontext
def shutdown_session(exception=None):
    db_session.remove()

if __name__ == '__main__':
  app.run(debug=True,host='0.0.0.0',port=8000)
