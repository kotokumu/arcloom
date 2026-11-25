"""Example trusted execution driver: JSON in, JSON out; no provider credentials."""
import json
import sys

request = json.load(sys.stdin)
payload = request["input"]
text = payload["text"]
operation = payload["operation"]
if not isinstance(text, str):
    raise ValueError("text must be a string")
operations = {"upper": str.upper, "lower": str.lower, "title": str.title}
if operation not in operations:
    raise ValueError("unsupported text operation")
json.dump({"text": operations[operation](text)}, sys.stdout)
