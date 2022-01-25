import argparse
import datetime
import json
import logging

import requests
import yaml

logging.basicConfig(
    format="%(asctime)s %(message)s", datefmt="%Y-%m-%d %H:%M:%S", level=logging.INFO
)


def load_config():
    parser = argparse.ArgumentParser(description="Transaction auditor")
    parser.add_argument("-v", action="store_true", dest="verbose")
    parser.add_argument("--dryrun", action="store_true")
    args = vars(parser.parse_args())
    loc = ["./config.yaml", "/etc/auditor/config.yaml"]
    for path in loc:
        try:
            with open(path, "r") as f:
                c = yaml.safe_load(f)
                for k, v in args.items():
                    c[k] = v
                if args["verbose"]:
                    logging.info(f"Config loaded: {path}")
                return c
        except:
            pass

    raise Exception(f"Config not found in: {', '.join(loc)}")


CONFIG = load_config()


def check_outstanding(description, from_):
    if CONFIG["verbose"]:
        logging.info(f"Checking outstanding: {description}")
    result = []
    params = {
        "description__like": description,
        "created__gt": (
            datetime.datetime.now() - datetime.timedelta(CONFIG["filters"].get("days"))
        ).strftime("%Y-%m-%dT%H:%M:%S"),
    }
    req = requests.get("https://moneyman.k3s.salinas.id.au/api/transactions", params)
    # print(req.request.url, req.json())
    data = req.json()
    for transaction in data["data"]:
        p = {
            "amount": "{:0.2f}".format(abs(transaction["amount"])),
            "created__gt": transaction["created"][0:19],
            "description__like": from_,
        }
        # print(f"Looking for {p['amount']} after {p['created__gt']}")
        r = requests.get("https://moneyman.k3s.salinas.id.au/api/transactions", p)
        d = r.json()["data"]
        if len(d) == 0:
            dd = datetime.datetime.strptime(transaction["created"][0:10], "%Y-%m-%d")
            result.append(
                "${:0.2f} from {}".format(
                    abs(transaction["amount"]), dd.strftime("%a %-d %b")
                )
            )

    return result


def run_checks():

    for k, v in CONFIG["checks"].items():
        r = check_outstanding(k, v["from"])
        if r:
            m = f"Move money from {v['from']} to {v['to']}:\n"
            m += "\n".join(r)
            if not CONFIG["dryrun"]:
                notify(m)
            if CONFIG["verbose"]:
                logging.info(f"Found {len(r)} outstanding transactions")


def notify(message):
    headers = {
        "Content-Type": "application/x-www-form-urlencoded",
        "Accept": "application/json",
    }
    for m in CONFIG["notify"]["mobiles"]:
        data = {"From": "Budget", "To": m, "Body": message}
        resp = requests.post(
            f"https://api.twilio.com/2010-04-01/Accounts/{CONFIG['notify']['sid']}/Messages",
            data=data,
            auth=(CONFIG["notify"]["sid"], CONFIG["notify"]["token"]),
        )
        if CONFIG["verbose"]:
            logging.info(f"Sent SMS")


if __name__ == "__main__":
    run_checks()
