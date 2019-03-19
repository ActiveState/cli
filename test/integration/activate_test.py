import os
import helpers
import auth_test

dir_path = os.path.abspath(os.path.dirname(os.path.realpath(__file__)))

class TestActivate(helpers.IntegrationTest):

    def test_activate(self):
        self.spawn("")
        self.expect("Usage:")
        self.assertEqual(0, self.wait(), "exits with code 0")

    def test_activate_project(self):
        path = os.path.join(dir_path, "testdata")
        self.set_cwd(path)

        self.spawn("activate")
        self.expect("Login with my existing account")
        self.send_quit()
        self.wait()
        
        auth = auth_test.TestAuth()
        auth.set_config(self.config_dir)
        auth.set_cwd(path)
        auth.auth_signup()
        
        self.spawn("activate")
        self.expect("activated")
        self.send_quit()
        self.wait()

if __name__ == '__main__':
    helpers.Run()