import os
import sys
import re
import subprocess
import tempfile

import auth_test
import helpers

testdir = os.path.abspath(os.path.dirname(os.path.realpath(__file__)))
projectdir = os.path.abspath(os.path.dirname(os.getenv("ACTIVESTATE_PROJECT")))

class TestUpdates(helpers.IntegrationTest):

    def __init__(self, *args, **kwargs):
        super(TestUpdates, self).__init__(*args, **kwargs)
        self.constants = {}
        self.parseConstantsFiles()

    def getPlatform(self):
        plat = ""
        arch = "amd64"
        if sys.platform == "win32":
            plat = "windows"
        else:
            plat = sys.platform # linux and darwin are output on those platforms
        return plat+"-"+arch

    def parseConstantsFiles(self):
        constPath = os.path.join(projectdir, "internal", "constants", "generated.go")
        goConstVarRe = re.compile(
            "const\s+(?P<name>[\w\d]+)\s+=\s\"(?P<value>.*?)\"")
        with open(constPath, 'r') as f:
            for line in f:
                match = goConstVarRe.search(line)
                if match != None:
                    self.constants[match.group("name")] = match.group("value")

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
        archivePath = os.path.join(projectdir,
                                   "public",
                                   "update",
                                   self.constants["BranchName"],
                                   self.constants["Version"],
                                   platform+self.getArchExt())
        
        if platform.startswith("windows"):
            unArchiveCmd = ["powershell.exe",
                            "-nologo",
                            "-noprofile",
                            "-command",
                            "\"Expand-Archive '{0}' (Get-Location)\"".format(archivePath)]
        else:
            unArchiveCmd = ["tar",
                            "-C",
                            ".",
                            "-xf",
                            archivePath]
            
        done = subprocess.run(unArchiveCmd)
        self.assertEqual(0, done.returncode, "Nothing should go wrong")
        cmd = "{0} --version".format(os.path.join(testdir,platform+self.getBinExt()))
        self.spawn_command(cmd)
        self.expect(self.constants["BuildNumber"])


if __name__ == '__main__':
    helpers.Run()