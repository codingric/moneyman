import requests
import json
import sys

accounts = {"Spending": "62863432", "BigBills": "37366510"}


def main():
    data = json.load(sys.stdin)

    items = sorted(data["Items"], key=lambda x: x["When"]["S"])

    for item in items:
        try:
            amount = "{:0.2f}".format(float(item["Payload"]["M"]["amount"]["S"]) * -1)
            account = (
                accounts[item["Payload"]["M"]["from"]["S"]]
                if item["Payload"]["M"]["type"]["S"] == "payment"
                else ""
            )
            payload = {
                "amount": amount,
                "description": item["Payload"]["M"]["detail"]["S"],
                "account": account,
                "created": item["When"]["S"][0:19] + "+00:00",
            }

            rep = requests.post("http://localhost:8080/transactions", json=payload)
            if rep.status_code != 200:
                print(f"Payload: {payload}")
                print(rep.text)
                print("---")
        except Exception:
            pass


if __name__ == "__main__":
    main()