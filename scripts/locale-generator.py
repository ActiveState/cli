import subprocess
import re
import os

os.chdir(os.path.join(os.path.dirname(__file__), ".."))
result = subprocess.check_output(['git', 'diff'])
matches = re.findall(re.compile(r"^\+.*locale.Tr?\(.(\w+)", re.M), result.decode('utf-8'))

with open('internal/locale/en-us.yaml', 'r') as file:
    contents = file.read()

with open('internal/locale/en-us.yaml', 'a') as file:
    for match in matches:
        if not re.search(re.compile("^%s:" % match, re.M), contents):
            print("Adding: %s" % match)
            file.write("\n%s:\n  other: TODO" % match)
