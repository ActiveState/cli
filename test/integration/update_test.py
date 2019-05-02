import os
import sys
import re
import subprocess
import tempfile

import auth_test
import helpers


class TestUpdates(helpers.IntegrationTest):
    S3_UPDATE_URL = "https://s3.ca-central-1.amazonaws.com/cli-update/update/"
    CONSTANTS = {}

    def __init__(self, *args, **kwargs):
        super(TestUpdates, self).__init__(*args, **kwargs)
        self.parseConstants()

    def getPlatform(self):
        plat = ""
        arch = "amd64"
        if sys.platform == "linux2":
            plat = "linux"
        if sys.platform == "win32":
            plat = "windows"
        else:
            plat = - "darwin"
        return plat+"-"+arch

    def parseConstants(self):
        constPath = os.path.join("..", "internal", "constants", "generated.go")
        goConstVarRe = re.compile(
            "const\s+(?P<name>\w+)\s+=\s(?P<value>\"*\w+\"*)")
        with open(constPath, 'r') as f:
            for line in f:
                match = goConstVarRe.search(line)
                if match != None:
                    self.CONSTANTS[match.group("name")] = match.group("value")

    def getArchExt(self):
        if sys.platform == "win32":
            return ".zip"
        else:
            return ".tar.gz"

    def getBinExt(self):
        if sys.platform == "win32":
            return ".exe"
        else:
            return ""

    def TestUpdatesWorks(self):
        unArchiveCmd = ""
        platform = self.getPlatform()
        archivePath = os.path.join(
            "..", "..", "public", "update", self.CONSTANTS["BranchName"], self.CONSTANTS["VERSION"], platform+self.getArchExt())
        if platform.startswith("windows"):
            unArchiveCmd = "powershell.exe -nologo -noprofile -command \"Expand-Archive '{0}' (Get-Location)\"".format(
                archivePath)
        else:
            unArchiveCmd = "tar -xf {0}".format(archivePath)
        done = subprocess.run(unArchiveCmd)
        if done.returncode != 0:
            pass
        self.spawn("{0} --version".format(platform+self.getBinExt()))
        self.expect(self.CONSTANTS["Version"])
