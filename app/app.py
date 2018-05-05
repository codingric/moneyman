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
from  dateutil.relativedelta import relativedelta

app = Flask(__name__)
app.secret_key = "lkaskdvmiiv887773n3nmnvfdv"
app.config['DEBUG'] = True

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
    start_date = date.today() + relativedelta(months=-1)
    sql = text("""
      SELECT
        t.tag as tag,
        CAST(sum(t.amount) AS FLOAT) as total,
        CAST(ifnull(budget, 0) AS FLOAT) as budget,
        CAST(abs(ifnull(budget, 0))-abs(sum(t.amount)) AS FLOAT) as variance
      FROM transactions t
      LEFT JOIN monthly_budget b ON t.tag = b.tag
      WHERE t.date > '{start_date}'
      GROUP BY t.tag
      ORDER BY 2
      """.format(start_date=start_date.isoformat()))
    stats = db_session.execute(sql)
    return render_template("index.html", stats = stats, start_date=start_date)


@app.route('/tags')
def tags():
  return render_template('tag.html', tags=Tag.query.order_by(Tag.name.asc()).all())

@app.route('/tags', methods=['POST'])
def post_tags():

  if not request.form['tag'] or not request.form['name']:
    flash('Required fields missing', 'error')
    return redirect(url_for('tags'))

  b = request.form.get('budget_amount')
  f = request.form.get('budget_frequency')
  b = float(b)
  f = FrequencyEnum[f]
  
  obj = Tag(
    name=request.form['name'],
    tag=request.form['tag'],
    budget_amount=b,
    budget_frequency=f,
    description=request.form['description']
  )
  
  db_session.add(obj)
  db_session.commit()

  flash('New tag created', 'ok')
  return redirect(url_for('tags'))


@app.route('/tag/<int:tag_id>', methods=['POST'])
def tag(tag_id):

  t = Tag.query.get(tag_id)
  data = { k: v for k, v in request.form.to_dict().iteritems() if hasattr(t, k) }

  if request.form.get('_method','').upper() == 'PATCH':
    for k, v in data.iteritems():
      if k == "budget_amount":
        v = float(v)
      print "UPDATING: ", k, v
      setattr(t, k, v)
  
    db_session.add(t)
    db_session.commit()
    return redirect(url_for('tags'))

  if request.form.get('_method','').upper() == 'DELETE':
    t = Tag.query.get(tag_id)
    if not t:
      return 'Tag not found', 404
    db_session.delete(t)
    db_session.commit()
    return redirect(url_for('tags'))
  return 'Not allowed', 405

@app.route('/transactions', methods=['GET', 'POST'])
def transactions():
  if request.method == 'POST':
    t = Transaction.query.get(int(request.form.get('trans_id')))
    t.tag = None
    newtag = request.form.get('tag')
    if newtag != '':
      t.tag = newtag
    
    db_session.add(t)
    db_session.commit()
    print(url_for('transactions', _anchor="trans%d"%t.id))
    return redirect(url_for('transactions', tag=request.args.get('tag'),_anchor="trans%d"%t.id))
  tag_filter = "(t.tag IS NULL OR t.tag != 'Exclude')"

  if request.args.get('tag'):
    if request.args.get('tag') == '__Empty__':
      tag_filter = "t.tag IS NULL"
    else: 
      tag_filter = "t.tag = '{}'".format(request.args.get('tag'))

  start_date = date.today() + relativedelta(months=-1)
  sql = text("""
    SELECT
      t.*,
      a.name as account_name
    FROM transactions t
    JOIN accounts a ON a.number = t.account
    WHERE date > '{start_date}'
      AND {tag_filter}
    ORDER BY date DESC""".format(tag_filter=tag_filter, start_date=start_date))
  transactions = db_session.execute(sql)
  return render_template('transactions.html', transactions=transactions, start_date=start_date)

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

    matchers = [ (re.compile(m[0].encode('utf-8')), m[1], m[2]) for m in db_session.query(Matcher.regex,Matcher.account,Matcher.tag).all() ]

    i = Importer(INGDIRECT_CONFIG, file_path="./import.csv")
    imported = 0
    for row in i.read():
      if row['ref'] == ref:
        break
      transaction = Transaction(**row)
      transaction.tag = find_match(transaction, matchers)
      db_session.add(transaction)
      imported += 1

    db_session.commit()
    flash("Imported %d new transactions" % imported)
    return redirect(url_for('transactions'))
  return render_template('upload.html')

def find_match(t, m):
  for r, a, v in m:
    if (a==None or t.account==a) and r.match(t.description.encode('utf-8')):
      print('match %s %s' % (t, v))
      return v
  return None

@app.teardown_appcontext
def shutdown_session(exception=None):
    db_session.remove()

if __name__ == '__main__':
  app.run(debug=True,host='0.0.0.0',port=8000)
