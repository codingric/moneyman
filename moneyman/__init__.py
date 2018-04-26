import re
import csv
import hashlib
import datetime

INGDIRECT_CONFIG = {
  "date": lambda r: datetime.datetime.strptime(r['Date'], "%d/%m/%Y"),
  "description": lambda r: str(re.search("^.*(?= - Receipt)", r['Description']).group(0) if ' - Receipt ' in r['Description'] else r['Description']).decode("utf-8", "ignore"),
  "ref": lambda r: hashlib.md5("%s %s %s" % (r['Date'], r['Account'], r['Description'])).hexdigest(),
  "amount": lambda r: float(r['Credit'] if r['Credit'] else r['Debit']),
  "account": "Account"
}

class Importer:

  def __init__(self, config, stream = None, file_path = None):
    self.config = config
    
    if stream:
      self.stream = stream

    if file_path:
      self.stream = open(file_path, "r")

  def read(self):
    reader = csv.DictReader(self.stream)
    for row in reader:
      yield self.parse_row(row)

  def parse_row(self, row):
    result = {}
    for key, value in self.config.iteritems():
      if callable(value):
        result[key] = value(row)
      else:
        result[key] = row.get(value, None)
    return result

  def match_tag(tags, description):
    for k, v in tags.iteritems():
      if re.match(v, description):
        return k
