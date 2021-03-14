# Download the helper library from https://www.twilio.com/docs/python/install
import os
from twilio.rest import Client
import logging

logger = logging.getLogger()
logger.setLevel(logging.INFO)

# Your Account Sid and Auth Token from twilio.com/console
# and set the environment variables. See http://twil.io/secure
account_sid = os.environ["TWILIO_ACCOUNT_SID"]
auth_token = os.environ["TWILIO_AUTH_TOKEN"]

client = Client(account_sid, auth_token)


def handler(event, context):
    logger.info("event: %s", event)

    text = f'{event["detail"]["type"].title()} from {event["detail"]["from"]} to {event["detail"].get("to") or event["detail"]["detail"]} for ${event["detail"]["amount"]}'

    message = client.messages.create(
        body=text,
        from_=event.get("from", "Budget"),
        to="+61432071731",
    )
    logger.info("message.id: %s", message.sid)
