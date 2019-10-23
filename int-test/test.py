from pexpect import popen_spawn as p


def main():

    cmd = ".\\build\\state.exe auth"
    print("spawning")
    child = p.PopenSpawn(cmd)
    child.logfile = Logger(cmd)
    print("expecting 1")
    child.expect("username:")
    print("sending 1")
    child.sendline(b"abc")
    print("expecting 2")
    child.expect("password:")
    print("Done")


class Logger:

    def __init__(self, cmd):
        self.logfile = open("log.log", "wb")
        self.logfile.write(("-- Executing '%s' --\n\n" % cmd).encode())
        self.logged = ""

    def write(self, s):
        self.logfile.write(s)
        self.logged += s.decode()

    def flush(self):
        self.logfile.flush()


if __name__ == "__main__":
    main()