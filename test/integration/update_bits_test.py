import os
import sys
import re
import subprocess
import shutil

import helpers

test_dir = os.path.abspath(os.path.dirname(os.path.realpath(__file__)))
project_dir = os.path.realpath(os.path.join(test_dir, "..", ".."))


class TestUpdates(helpers.IntegrationTest):

    def __init__(self, *args, **kwargs):
        super(TestUpdates, self).__init__(*args, **kwargs)
        self.constants = helpers.get_constants()

    def get_platform(self):
        if sys.platform == "win32":
            return "windows" + "-" + "amd64"
        return sys.platform + "-" + "amd64"

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
                    "Expand-Archive -Path '{0}' -DestinationPath '{1}'".format(archive_path, self.temp_dir)]
        else:
            return ["tar",
                    "-C",
                    test_dir,
                    "-xf",
                    archive_path]

    def run_unarchive_cmd(self):
        platform = self.get_platform()
        done = subprocess.run(self.unarchive_cmd(platform))
        self.assertEqual(0, done.returncode, "Nothing should go wrong")

    def test_update_bits_work(self):
        self.run_unarchive_cmd()
        platform = self.get_platform()
        
        bin = os.path.join(test_dir, platform+self.get_bin_ext())
        cmd = "{0} --version".format(bin)
        self.spawn_command(cmd)
        self.expect(self.constants["BuildNumber"])
        self.wait()
        os.remove(bin)

if __name__ == '__main__':
    helpers.Run()
