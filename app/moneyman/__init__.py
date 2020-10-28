import re
import csv
import hashlib
import datetime
import string

INGDIRECT_CONFIG = {
  "date": lambda r: datetime.datetime.strptime(r['Date'], "%d/%m/%Y"),
  "description": lambda r: ''.join([x if x in string.printable else '' for x in (str(re.search("^.*(?= - Receipt)", r['Description']).group(0) if ' - Receipt ' in r['Description'] else r['Description']))]),
  "ref": lambda r: hashlib.md5("%s %s %s".format(r['Date'], r['Account'], r['Description']).encode('utf-8')).hexdigest(),
  "amount": lambda r: float(r['Credit'] if r['Credit'] else r['Debit']),
  "account": lambda r: int(r["Account"])
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
    for key, value in self.config.items():
      if callable(value):
        result[key] = value(row)
      else:
        result[key] = row.get(value, None)
    return result

  def match_tag(tags, description):
    for k, v in tags.items():
      if re.match(v, description):
        return k
