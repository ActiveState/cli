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
            "const\s+(?P<name>\w+)\s*=\s*\"(?P<value>.*?)\"")
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

    def run_unarchive_cmd(self):
        platform = self.get_platform()
        done = subprocess.run(self.unarchive_cmd(platform))
        self.assertEqual(0, done.returncode, "Nothing should go wrong")

    def test_update_bits_work(self):
        self.run_unarchive_cmd()
        
        bin = os.path.join(test_dir, platform+self.get_bin_ext())
        cmd = "{0} --version".format(bin)
        self.spawn_command(cmd)
        self.expect(self.constants["BuildNumber"])
        self.wait()
        os.remove(bin)

    def get_version_from_output(self, output):
        version_regex = re.compile(".*(\d\.\d\.\d-\d{4})")
        match = version_regex.search(str(output))
        if match:
            return match.group(1)

    def _assert_version(self, same, bin_path):
        shutil.copy(self.get_build_path(), test_dir)
        version = self.get_version_from_output(self.get_output("%s --version" %(bin_path)))
        if same:
            self.assertEqual(version, self.constants["Version"], "They should be equal.")
        else:
            self.assertNotEqual(version, self.constants["Version"], "They should not be equal.")
        os.remove(bin_path)

    def test_update_works(self):
        # get the binary
        bin_path = os.path.join(test_dir, self.get_binary_name())
        # Turn enable updates in tests
        self.env["ACTIVESTATE_CLI_DISABLE_UPDATES"] = "false"
        # run state --version
        # check version changed
        self._assert_version(False, bin_path)
        #run state update
        # confirm version changed
        self.spawn_command("%s update" %(bin_path))
        self.wait()
        self._assert_version(False, bin_path)
        # set versionlock `state update --lock`
        # run --version
        # Verions doesn't change
        self.spawn_command("%s update --lock" %(bin_path))
        self.wait()
        self._assert_version(True, bin_path)

if __name__ == '__main__':
    helpers.Run()
