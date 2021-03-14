import boto3
import os
import json
import logging
import sys
import re
import datetime
import uuid

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)


def send_event(payload):
    client = boto3.client("events")
    entries = [
        {
            "EventBusName": os.environ["EVENTBUS_NAME"],
            "DetailType": "Transaction",
            "Detail": json.dumps(payload),
            "Source": "Webhook",
        }
    ]
    logger.debug("Entries:" + str(entries))
    return client.put_events(Entries=entries)


def save(payload):
    try:
        dc = boto3.client("dynamodb")
        dc.put_item(
            TableName=os.environ["DYNAMOTABLE_NAME"],
            Item={
                "RequestID": {"S": str(uuid.uuid4())},
                "Payload": dict_to_item(payload),
                "When": {"S": datetime.datetime.today().astimezone().isoformat()},
            },
        )
    except Exception as e:
        logger.info("Payload:" + str(payload))
        logger.exception(e, exc_info=True)


def dict_to_item(raw):
    if isinstance(raw, dict):
        return {"M": {key: dict_to_item(value) for key, value in raw.items()}}
    elif isinstance(raw, list):
        return {"L": [dict_to_item(value) for value in raw]}
    elif isinstance(raw, (str)):
        return {"S": raw}
    elif isinstance(raw, bool):
        return {"BOOL", raw}
    elif isinstance(raw, (int, float)):
        return {"N": str(raw)}
    elif isinstance(raw, bytes):
        return {"B", raw}
    elif raw is None:
        return {"NULL": True}


def detect_transaction(text, assumed_account):
    payload = {"type": "unknown"}

    transfer = re.search(
        "(?s)We're writing to let you know that [$](?P<amount>[0-9]+\.[0-9]+) was moved between your ING accounts, from (?P<from>.+?) to (?P<to>.+?)\.",
        text,
        re.M,
    )

    payment = re.search(
        "(?s)We're writing to confirm your recent payment of [$](?P<amount>[0-9]+\.[0-9]+) to (?P<detail>[^.]+?)\.",
        text,
        re.M,
    )

    if not payment:
        payment = re.search(
            "(?s)We're writing to let you know you recently spent [$](?P<amount>[0-9]+\.[0-9]+) at (?P<detail>[^.]+?)\.",
            text,
            re.M,
        )

    deposit = re.search(
        "We're writing to let you know that (?P<detail>.+?) deposited [$](?P<amount>[0-9]+\.[0-9]+) into your (?P<to>[a-zA-Z ]+) account\.",
        text,
        re.M,
    )

    if not deposit:
        deposit = re.search(
            "We're writing to let you know that a [$](?P<amount>[0-9]+\.[0-9]+) deposit has been made into your (?P<to>.+?) account\.",
            text,
            re.M,
        )

    if transfer:
        payload = transfer.groupdict()
        payload["type"] = "transfer"

    if payment:
        payload = payment.groupdict()
        payload["type"] = "payment"
        payload["from"] = assumed_account

    if deposit:
        payload = deposit.groupdict()
        payload["type"] = "deposit"

    payload["when"] = datetime.date.today().isoformat()

    if payload["type"] == "unknown":
        payload["original"] = text
    else:
        if not "from" in payload:
            payload["from"] = "unknown"

    logger.debug(payload)
    return payload


def assumed_account(email):
    m = {
        "c77bce21e0a9d86c7077@cloudmailin.net": "BigBills",
        "2c445d5ae86004faa078@cloudmailin.net": "Spending",
    }
    return m[email]


def extract_wording(html):
    parsed = re.sub("(?s)<.*?>", "", html)
    parsed = re.sub("(?s)\s+", " ", parsed)
    logger.info("extract_wording: %s", parsed)
    return parsed


def handler(event, context):
    logger.info("event:" + str(event))
    body = json.loads(event["body"])
    wording = extract_wording(body["html"])
    assumed = assumed_account(body["envelope"]["to"])
    transaction = detect_transaction(wording, assumed)
    save(transaction)
    logger.info("transaction:" + str(transaction))
    logger.info(send_event(transaction))
    return {"body": json.dumps(transaction), "statusCode": 200}


if __name__ == "__main__":
    logging.basicConfig(
        stream=sys.stdout,
        level=logging.INFO,
        format="%(levelname)s %(filename)s:%(funcName)s:%(lineno)d %(message)s",
    )
    sys.path.insert(0, "/Users/ricardo/src/moneyman/lambdas")
    from payloads import payment

    logger.info("--- RESPONSE ---\n%s", handler(payment, {}))
