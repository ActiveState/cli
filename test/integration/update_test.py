import os
import sys
import re
import subprocess

import helpers

test_dir = os.path.abspath(os.path.dirname(os.path.realpath(__file__)))
project_dir = os.path.realpath(os.path.join(test_dir, "..", ".."))


class TestUpdates(helpers.IntegrationTest):

    def __init__(self, *args, **kwargs):
        super(TestUpdates, self).__init__(*args, **kwargs)
        self.constants = {}
        self.parse_constants_files()

    def get_platform(self):
        if sys.platform == "win32":
            return "windows" + "-" + "amd64"
        return sys.platform + "-" + "amd64"

    def parse_constants_files(self):
        const_path = os.path.join(
            project_dir, "internal", "constants", "generated.go")
        go_const_var_re = re.compile(
            "const\s+(?P<name>\w+)\s*+=\s*\"(?P<value>.*?)\"")
        with open(const_path, 'r') as f:
            for line in f:
                match = go_const_var_re.search(line)
                if match != None:
                    self.constants[match.group("name")] = match.group("value")

    def get_arch_ext(self):
        if sys.platform == "win32":
            return ".zip"
        return ".tar.gz"

    def get_bin_ext(self):
        if sys.platform == "win32":
            return ".exe"
        return ""

    def unarchive_cmd(self, platform):
        archive_path = os.path.join(project_dir,
                                    "public",
                                    "update",
                                    self.constants["BranchName"],
                                    self.constants["Version"],
                                    platform+self.get_arch_ext())

        if platform.startswith("windows"):
            return ["powershell.exe",
                    "-nologo",
                    "-noprofile",
                    "-command",
                    "\"Expand-Archive -LiteralPath '{0}' -DestinationPath '{1}'\"".format(archive_path, test_dir)]
        else:
            return ["tar",
                    "-C",
                    test_dir,
                    "-xf",
                    archive_path]

    def test_update_works(self):
        platform = self.get_platform()
        done = subprocess.run(self.unarchive_cmd(platform))
        self.assertEqual(0, done.returncode, "Nothing should go wrong")

        cmd = "{0} --version".format(os.path.join(test_dir,
                                                  platform+self.get_bin_ext()))
        self.spawn_command(cmd)
        self.expect(self.constants["BuildNumber"])
        self.wait(code=0)


if __name__ == '__main__':
    helpers.Run()
