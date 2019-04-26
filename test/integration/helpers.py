import os
import shutil
import signal
import tempfile
import time
import unittest
import uuid
import warnings
import pexpect
import requests
from pexpect.popen_spawn import PopenSpawn
import psutil

is_windows = os.name == 'nt'

spawner = pexpect.spawn
if is_windows:
    spawner = PopenSpawn

dir_path = os.path.abspath(os.path.dirname(os.path.realpath(__file__)))
try:
    os.remove(os.path.join(dir_path, "integration.log"))
except FileNotFoundError:
    pass

class IntegrationTest(unittest.TestCase):

    def __init__(self, *args, **kwargs):
        super(IntegrationTest, self).__init__(*args, **kwargs)
        self.cwd = None
        self.child = None

    def setUp(self):
        # Disable resource warnings because pexpect doesn't seem to clean up its threads properly and that's not our problem
        warnings.filterwarnings("ignore", category=ResourceWarning)
        self.clear_config()
        self.clear_cache()

    def tearDown(self):
        time.sleep(0.1) # Required to ensure the child process has had time to quit

        if self.is_running():
            self.send_quit()
            self.fail("Command is still running after test, sent QUIT signal to %d" % self.pid())

    def pid(self):
        if is_windows:
           return self.child.proc.pid
        else:
           return self.child.ptyproc.pid

    def spawn(self, args):
        dir_path_top = os.path.join(dir_path, "..", "..")
        self.spawn_command('%s/build/state %s' % (dir_path_top, args))

    def spawn_command(self, cmd):
        self.child = spawner(cmd, env=self.env, timeout=10)
        self.child.logfile_read = IntegrationLogger(cmd)

    def clear_config(self):
        self.set_config(tempfile.mkdtemp())

    def clear_cache(self):
        cache_dir = os.path.expanduser("~/.cache/activestate/cli")
        if os.path.isdir(cache_dir):
            shutil.rmtree(cache_dir)

    def set_config(self, config_dir):
        self.config_dir = config_dir
        self.env = os.environ.copy()
        self.env["ACTIVESTATE_CLI_CONFIGDIR"] = config_dir
        #print("%s is using configdir: %s" % (self.id(), config_dir))

    def set_cwd(self, cwd):
        self.cwd = cwd
        os.chdir(cwd)

    def expect(self, pattern, timeout=2):
        try:
            idx = self.child.expect(pattern, timeout=timeout)
        except pexpect.EOF:
            self.send_quit()
            self.expect_failure("Reached EOF", pattern)
        except pexpect.TIMEOUT:
            self.send_quit()
            raise self.expect_failure("Reached timeout", pattern)

    def expect_exact(self, pattern, timeout=2):
        try:
            idx = self.child.expect_exact(pattern, timeout=timeout)
        except pexpect.EOF:
            self.send_quit()
            self.expect_failure("Reached EOF", pattern)
        except pexpect.TIMEOUT:
            self.send_quit()
            raise self.expect_failure("Reached timeout", pattern)

    def expect_failure(self, message, pattern):
        self.fail("%s while expecting '%s', output:\n---\n%s\n---" % (message, pattern, self.child.logfile_read.logged))

    def send(self, message):
        self.child.sendline(message)

    def send_quit(self):
        if self.is_running():
            os.kill(self.pid(), signal.SIGQUIT)
        if not is_windows:
            self.child.close()

    def is_running(self):
        if not self.child:
            return False

        try:
            status = psutil.Process(self.pid()).status()
        except psutil.NoSuchProcess:
            return False
        return status == "running"

    def wait_ready(self, timeout=10):
        self.send("echo wait_ready_$HOME")
        self.expect_exact("wait_ready_%s" % os.path.expanduser("~"), timeout=timeout)

    def wait(self, code=0, timeout=10):
        try:
            with wait_for_timeout(seconds=timeout):
                result = self.child.wait()
        except TimeoutError:
            self.fail("timeout while waiting, output:\n---\n%s\n---" % (self.child.logfile_read.logged))
            return

        result = result or 0
        self.assertEqual(code, result, "exits with code %d, output:\n---\n%s\n---" % (code, self.child.logfile_read.logged))
        return result

    def fail(self, msg=None):
        """Fail immediately, with the given message."""
        raise self.failureException(msg) from None

class IntegrationLogger:

    def __init__(self, cmd):
        self.logfile = open(os.path.join(dir_path, "integration.log"), "ab")
        self.logfile.write(("-- Executing '%s' --\n\n" % cmd).encode())
        self.logged = ""

    def write(self, s):
        self.logfile.write(s)
        self.logged += s.decode()

    def flush(self):
        self.logfile.flush()

class wait_for_timeout:
    def __init__(self, seconds=1, error_message='Timeout'):
        self.seconds = seconds
        self.error_message = error_message
    def handle_timeout(self):
        raise TimeoutError(self.error_message)
    def __enter__(self):
        signal.signal(signal.SIGALRM, self.handle_timeout)
        signal.alarm(self.seconds)
    def __exit__(self, type, value, traceback):
        signal.alarm(0)

def Run():
    unittest.main()