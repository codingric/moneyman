# BigBills

This application connects to BigBills spreadsheet using google api, checks for outstanding BigBills and notifies.

## Google credentials

Requires a credentials.json that allows to access the Bugbet spreadsheet

## Twillio config

Requires a config.yaml with follow config:

```
sid: <twillio_account_sid>
token: <twillio_account_token>
mobiles:
  - <+614xxxxxxxx>
```