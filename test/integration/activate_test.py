import os	
import tempfile	

 import auth_test	
import helpers	

 dir_path = os.path.abspath(os.path.dirname(os.path.realpath(__file__)))	

 class TestActivate(helpers.IntegrationTest):	

     def test_activate_python2(self):	
        path = os.path.join(dir_path, "testdata")	
        self.set_cwd(path)	

         auth = auth_test.TestAuth()	
        auth.set_config(self.config_dir)	
        auth.set_cwd(path)	
        auth.login_as_persistent_user()	

         self.env["ACTIVESTATE_CLI_DISABLE_RUNTIME"] = "false"	

         self.spawn("activate ActiveState-CLI/Python2")	
        self.expect("Where would you like to checkout")	
        self.send(tempfile.TemporaryDirectory().name)	
        self.expect("Downloading", timeout=120)	
        self.expect("Installing", timeout=120)	
        self.wait_ready(timeout=120)	
        self.send("python2 -c 'import sys; print(sys.copyright)'")	
        self.expect("ActiveState Software Inc.")	
        self.send_quit()	
        self.wait()	

     def test_activate_python3(self):	
        path = os.path.join(dir_path, "testdata")	
        self.set_cwd(path)	

         auth = auth_test.TestAuth()	
        auth.set_config(self.config_dir)	
        auth.set_cwd(path)	
        auth.login_as_persistent_user()	

         self.env["ACTIVESTATE_CLI_DISABLE_RUNTIME"] = "false"	

         self.spawn("activate ActiveState-CLI/Python3")	
        self.expect("Where would you like to checkout")	
        self.send(tempfile.TemporaryDirectory().name)	
        self.expect("Downloading", timeout=120)	
        self.expect("Installing", timeout=120)	
        self.wait_ready(timeout=120)	
        self.send("python3 -c 'import sys; print(sys.copyright)'")	
        self.expect("ActiveState Software Inc.")	
        self.send_quit()	
        self.wait()	

 if __name__ == '__main__':	
    helpers.Run()