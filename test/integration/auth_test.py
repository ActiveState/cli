import time
import os

import helpers
import uuid
import requests

class TestAuth(helpers.IntegrationTest):

    def __init__(self, *args, **kwargs):
        super(TestAuth, self).__init__(*args, **kwargs)
        self.username = "user-%s" % uuid.uuid4().hex
        self.password = self.username
        self.email = "%s@test.tld" % self.username

    def test_auth(self):
        print("test_auth")
        if os.name == 'nt':
            return
        self.auth_signup()
        self.auth_logout()
        self.auth_login()

    def test_helpers(self):
        print("test_helpers")
        if os.name == 'nt':
            return
        self.auth_logout()
        self.login_as_persistent_user()
        self.auth_logout()
        self.create_user_and_login()

    def auth_signup(self):
        print("auth_signup")
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
        print("auth_logout")
        self.spawn("auth logout")
        self.expect("You have been logged out")
        self.wait()

    def auth_login(self):
        print("auth_login")
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

    def auth_login_with_flags(self):
        print("auth_login_with_flags")
        self.spawn("auth --username %s --password %s" % (self.username, self.password))
        self.expect("succesfully authenticated")
        self.wait()

        # still logged in?
        self.spawn("auth")
        self.expect("You are logged in")
        self.wait()

    def login_as_persistent_user(self):
        print("login_as_persistent_user")
        self.username = "cli-integration-tests"
        self.password = "test-cli-integration"
        self.auth_login_with_flags()

    def create_user_and_login(self):
        print("create_user_and_login")
        self.username, self.password = self.create_user()
        self.auth_login_with_flags()

    def create_user(self):
        print("create_user")
        username = "user-%s" % uuid.uuid4().hex
        data = {
            "name": username,
            "username": username,
            "password": username,
            "email": "%s@test.tld" % username,
            "EULAAccepted": True
        }
        r = requests.post("https://platform.activestate.com/api/v1/users", json=data)
        if r.status_code != 200:
            self.fail("Create user failed: code: %d, response: %s" % (r.status_code, r.content))
        return data["username"], data["password"]

if __name__ == '__main__':
    helpers.Run()