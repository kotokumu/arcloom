"""Example trusted feedback driver: compare actual result against Intent criteria."""
import json
import sys

request = json.load(sys.stdin)
payload = request["input"]
actual = payload["output"]["text"]
expected = payload["criteria"]["expected"]
if not isinstance(actual, str) or not isinstance(expected, str):
    raise ValueError("actual and expected must be strings")
json.dump({"accepted": actual == expected, "actual": actual, "expected": expected}, sys.stdout)
