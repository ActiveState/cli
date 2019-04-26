import helpers
import sys

class TestEnv(helpers.IntegrationTest):

    def test_project_env(self):
        self.spawn_command('%s -c "exec(\'import os\\nimport time\\ntime.sleep(1)\\nprint(os.environ[\\\'ACTIVESTATE_PROJECT\\\'])\')"' % sys.executable)
        self.expect("KeyError: 'ACTIVESTATE_PROJECT'")
        self.wait(code=1)

if __name__ == '__main__':
    helpers.Run()