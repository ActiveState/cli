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
import subprocess
import signal
import re

is_windows = os.name == 'nt'

spawner = None
if is_windows:
    spawner = PopenSpawn
else:
    spawner = pexpect.spawn

test_dir = os.path.abspath(os.path.dirname(os.path.realpath(__file__)))
try:
    os.remove(os.path.join(test_dir, "integration.log"))
except FileNotFoundError:
    pass

project_dir = os.path.realpath(os.path.join(test_dir, "..", ".."))

class IntegrationTest(unittest.TestCase):

    def __init__(self, *args, **kwargs):
        super(IntegrationTest, self).__init__(*args, **kwargs)
        self.cwd = None
        self.child = None

        self.env = os.environ.copy()
        self.env["ACTIVESTATE_CLI_DISABLE_UPDATES"] = "true"
        self.env["ACTIVESTATE_CLI_DISABLE_RUNTIME"] = "true"

        self.test_dir = test_dir
        self.project_dir = project_dir
        self.temp_dir = self.get_temp_path()

    def get_binary_name(self):
        if is_windows:
            return "state.exe"
        return "state"

    def get_build_path(self):
        return os.path.realpath(os.path.join(test_dir, "..", "..", "build", self.get_binary_name()))
    
    def get_temp_path(self):
        return os.path.join(tempfile.gettempdir(), uuid.uuid4().hex)

    def get_temp_bin(self):
        temp_bin = self.get_temp_path() + (".exe" if is_windows else "")
        shutil.copy(self.get_build_path(),  temp_bin)
        return temp_bin

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
        if self.env["ACTIVESTATE_CLI_DISABLE_UPDATES"] != "true":
            raise Exception("You're trying to run build/state while updates are enabled. This is not allowed. You should copy build/state somewhere and use spawn_command instead.")
        self.spawn_command('%s %s' % (self.get_build_path(), args))

    def spawn_command(self, cmd):
        self.child = spawner(cmd, env=self.env, timeout=10, cwd=self.cwd)
        self.child.logfile_read = IntegrationLogger(cmd)

    def spawn_command_blocking(self, cmd):
        if is_windows:
            cmd = cmd.replace("\\","/")
        args = pexpect.split_command_line(cmd)
        return subprocess.check_output(args, env=self.env, cwd=self.cwd, stderr=subprocess.DEVNULL)

    def clear_config(self):
        self.set_config(tempfile.mkdtemp())

    def clear_cache(self):
        cache_dir = os.path.expanduser("~/.cache/activestate")
        if is_windows:
            cache_dir = os.path.join(os.getenv("LOCALAPPDATA"),"activestate")
        if os.path.isdir(cache_dir):
            shutil.rmtree(cache_dir)

    def set_config(self, config_dir):
        self.config_dir = config_dir
        self.env["ACTIVESTATE_CLI_CONFIGDIR"] = config_dir
        #print("%s is using configdir: %s" % (self.id(), config_dir))

    def set_cwd(self, cwd):
        self.cwd = cwd
        os.chdir(cwd)

    def reset_cwd(self):
        self.cwd = None
        os.chdir(self.test_dir)

    def expect(self, pattern, timeout=10):
        try:
            idx = self.child.expect(pattern, timeout=timeout)
        except (pexpect.EOF, pexpect.exceptions.EOF):
            self.send_quit()
            self.expect_failure("Reached EOF", pattern)
        except (pexpect.TIMEOUT, pexpect.exceptions.TIMEOUT):
            self.send_quit()
            raise self.expect_failure("Reached timeout", pattern)

    def expect_exact(self, pattern, timeout=10):
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
            if is_windows:
                self.child.proc.terminate()
            else:
                os.kill(self.pid(), signal.NSIG)
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

    def wait_ready(self, timeout=30):
        msg = "echo wait_ready_"+os.path.expanduser("~")
        self.send(msg)
        self.expect_exact(msg, timeout=timeout)

    def wait(self, code=0, timeout=30):
        try:
            if is_windows:
                result = _win_wait_for_timeout(seconds=timeout, func=self.child.wait).wait()
            else:
                with _unix_wait_for_timeout(seconds=timeout):
                    result = self.child.wait()
        except TimeoutError:
            self.fail("timeout while waiting, output:\n---\n%s\n---" % (self.child.logfile_read.logged))
            return
        result = result or 0
        self.assertEqual(code, result, "exits with code %d, output:\n---\n%s\n---" % (code, self.child.logfile_read.logged))
        return result

    def fail(self, msg=None):
        """Fail immediately, with the given message."""
        raise self.failureException(msg)

class IntegrationLogger:

    def __init__(self, cmd):
        self.logfile = open(os.path.join(test_dir, "integration.log"), "ab")
        self.logfile.write(("-- Executing '%s' --\n\n" % cmd).encode())
        self.logged = ""

    def write(self, s):
        self.logfile.write(s)
        self.logged += s.decode()

    def flush(self):
        self.logfile.flush()

class _win_wait_for_timeout:
    from multiprocessing.pool import ThreadPool
    pool = ThreadPool(processes=1)
    import time
    def __init__(self, seconds=1, error_message='Timeout', func=lambda:1, args=None):
        self.seconds = seconds
        self.func = func
        self.result = None
        self.error_message = error_message

    def wait(self):
        def callback(out):
            self.result = out
        self.pool.apply_async(self.func, callback=callback)
        time.sleep(self.seconds)
        if self.result is None:
            raise TimeoutError(self.error_message)
        return self.result

class _unix_wait_for_timeout:
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

def get_constants():
        const_path = os.path.join(
            project_dir, "internal", "constants", "generated.go")
        go_const_var_re = re.compile(
            "const\s+(?P<name>\w+)\s*=\s*\"(?P<value>.*?)\"")
        constants = {}
        with open(const_path, 'r') as f:
            for line in f:
                match = go_const_var_re.search(line)
                if match != None:
                    constants[match.group("name")] = match.group("value")
        return constants

def Run():
    unittest.main()
