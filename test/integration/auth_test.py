import time
import helpers

class TestAuth(helpers.IntegrationTest):

    def __init__(self, *args, **kwargs):
        super(TestAuth, self).__init__(*args, **kwargs)
        self.username = "user-%d" % time.time()
        self.password = self.username
        self.email = "%s@test.tld" % self.username

    def auth_signup(self):
        self.spawn("auth signup")
        self.expect("username:")
        self.send(self.username)
        self.expect("password:")
        self.send(self.password)
        self.expect("again:")
        self.send(self.password)
        self.expect("name:")
        self.send(self.username)
        self.expect("email:")
        self.send(self.email)
        self.expect("account has been registered")
        self.wait()

    def auth_logout(self):
        self.spawn("auth logout")
        self.expect("You have been logged out")
        self.wait()

    def auth_login(self):
        self.spawn("auth")
        self.expect("username:")
        self.send(self.username)
        self.expect("password:")
        self.send(self.password)
        self.expect("succesfully authenticated")
        self.wait()

        # still logged in?
        self.spawn("auth")
        self.expect("You are logged in")
        self.wait()

    def test_auth(self):
        self.auth_signup()
        self.auth_logout()
        self.auth_login()

if __name__ == '__main__':
    helpers.Run()