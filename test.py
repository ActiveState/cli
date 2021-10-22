import json
import sys

entries = {}

for line in sys.stdin:
    data = json.loads(line)
    if data["Action"] != "pass":
        continue
    name = data["Package"]
    if "Test" in data:
        name += ":" + data["Test"]
    entries[name] = data["Elapsed"]

entriesSorted = dict(sorted(entries.items(), key=lambda item: item[1]))
for k in entriesSorted:
    print("%s: %.2f" % (k, entriesSorted[k]))
