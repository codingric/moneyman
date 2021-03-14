from __future__ import print_function
import pickle
import os.path
from googleapiclient.discovery import build
from google.oauth2 import service_account
import sys
import json
import datetime
import urllib3
import logging
import pytz
from pytz import timezone
import boto3

autz = timezone("Australia/Melbourne")


logger = logging.getLogger()
logger.setLevel(logging.INFO)

# If modifying these scopes, delete the file token.pickle.
SCOPES = ["https://www.googleapis.com/auth/spreadsheets"]

# The ID and range of a sample spreadsheet.
SPREADSHEET_ID = "1ieIu38LUKZVK24FAoNSjgVC6bQLeyD6PTbcZo_uIdig"
RANGE_NAME = "Big Bills!M2:N"


def extract_clientsecret():
    client = boto3.client("secretsmanager")
    response = client.get_secret_value(SecretId="/moneyman/google/clientsecret")
    os.environ["CLIENT_SECRET"] = response["SecretString"]


extract_clientsecret()

"""Shows basic usage of the Sheets API.
Prints values from a sample spreadsheet.
"""
credentials = service_account.Credentials.from_service_account_variable(
    "CLIENT_SECRET", scopes=SCOPES
)

service = build("sheets", "v4", credentials=credentials)


class ValueNotFound(Exception):
    pass


def get_big_bills():

    # Call the Sheets API
    sheet = service.spreadsheets()
    result = (
        sheet.values().get(spreadsheetId=SPREADSHEET_ID, range=RANGE_NAME).execute()
    )
    values = result.get("values", [])
    logger.info(f"Extracted {len(values)} values")
    return values


def filter_by_date(values, date):
    for i, v in enumerate(values):
        if v[0].strip() == date:
            found = v[1][1:]
            logger.info(f"Found {date}: {found}")
            return (found, i)

    raise ValueNotFound(f"{date} not found in BigBills")


def move_money(amount):
    payload = {"value1": amount, "value2": "BigBills"}
    logger.info(f"IFTTT Post: {payload}")
    http = urllib3.PoolManager()
    request = http.request(
        "POST",
        "https://maker.ifttt.com/trigger/ING_Saver/with/key/cJIuhzloVLNEXIql4EfLLNc7DJj0XTBgXxi2p1dcyW1",
        body=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )
    response = request.data.decode("utf-8")
    logger.info("IFTTT Reponse: " + response)
    return response == "Congratulations! You've fired the ING_Saver event"


def update_big_bills(index):
    logger.info(f"Update: {index}")
    # Call the Sheets API
    sheet = service.spreadsheets()
    result = (
        sheet.values()
        .update(
            spreadsheetId=SPREADSHEET_ID,
            range=f"Big Bills!P{2+index}",
            valueInputOption="RAW",
            body={
                "values": [[datetime.datetime.now(autz).strftime("%d %b %y")]],
                "majorDimension": "ROWS",
            },
        )
        .execute()
    )
    logger.info(f"Update result: {result}")
    return True


def handler(event, context):
    logger.info(f"Handler: {event}, {context}")
    date = event.get("date", datetime.datetime.now(autz).strftime("%Y-%m-%d"))
    amounts = get_big_bills()
    amount, index = filter_by_date(amounts, date)
    if move_money(amount):
        update_big_bills(index)


if __name__ == "__main__":
    logger.info(handler(json.loads(sys.argv[1]) if len(sys.argv) > 1 else {}, {}))
