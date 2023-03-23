scripts:
  - name: activationMessage
    language: {{.Language}}
    value: |
      import textwrap
      print(textwrap.dedent("""
        Quick Start
        -----------
        * To add a package to your runtime, type "state install <package name>"
        * Learn more about how to use the State Tool, type "state learn"
      """))
  - name: pip
    language: {{.Language}}
    value: |
        import os
        import subprocess
        import sys

        env = os.environ.copy()
        env["ACTIVESTATE_SHIM"] = "pip"

        project_path = os.path.join(r"${project.path()}", "activestate.yaml")

        def configure_message():
            print("To configure this shim edit the following file:\n" + project_path + "\n")

        def mapcmds(mapping):
            for fromCmd, toCmd in mapping.items():
                if len(sys.argv) == 1:
                    print("pip requires an argument. Try:\n pip [install, uninstall, list, show, search, help]")
                    sys.exit()
                if sys.argv[1] != fromCmd:
                    continue

                print(("Shimming command to: 'state %s'") % toCmd)
                configure_message()

                code = subprocess.call(["state", toCmd] + sys.argv[2:], env=env)
                sys.exit(code)

        mapcmds({
            "help": "help",
            "install": "install",
            "uninstall": "uninstall",
            "list": "packages",
            "show": "info",
            "search": "search",
        })

        print("Could not shim your command as it is not supported by the State Tool.\n" + 
              "Please check 'state --help' to find the best analog for the command you're trying to run.\n")
        configure_message()

events:
  # This is the ACTIVATE event, it will run whenever a new virtual environment is created (eg. by running `state activate`)
  # On Linux and macOS this will be ran as part of your shell's rc file, so you can use it to set up aliases, functions, environment variables, etc.
  - name: ACTIVATE
    value: {{.LangExe}} $scripts.activationMessage.path()
