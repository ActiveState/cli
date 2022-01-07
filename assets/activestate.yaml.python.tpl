scripts:
  - name: activationMessage
    language: {{.Language}}
    value: |
      # -*- coding: utf-8 -*-
      import textwrap
      print(textwrap.dedent("""
        Quick Start
        ───────────
        • To add a package to your runtime, type "state install <package name>"
        • Learn more about how to use the State Tool, type "state learn"
      """))
  - name: pip
    language: {{.Language}}
    value: |
        import os
        import subprocess
        import sys

        env = os.environ.copy()
        env["ACTIVESTATE_SHIM"] = "pip"

        def mapcmds(mapping):
            for fromCmd, toCmd in mapping.items():
                if sys.argv[1] != fromCmd:
                    continue

                print(("Shimming command to 'state %s', to configure this shim edit the following file:\n" +
                       "${project.path()}/activestate.yaml\n") % toCmd)

                code = subprocess.call(["state", toCmd] + sys.argv[2:], env=env)
                sys.exit(code)

        mapcmds({
            "install": "install",
            "uninstall": "uninstall",
            "list": "packages",
            "show": "info",
            "search": "search",
        })

        print("Could not shim your command as it is not supported by the State Tool.\nPlease check 'state --help' to find " +
              "the best analog for the command you're trying to run.\n" +
              "To configure this shim edit the following file:\n${project.path()}/activestate.yaml\n")


events:
  # This is the ACTIVATE event, it will run whenever a new virtual environment is created (eg. by running `state activate`)
  # On Linux and macOS this will be ran as part of your shell's rc file, so you can use it to set up aliases, functions, environment variables, etc.
  - name: ACTIVATE
    value: {{.LangExe}} $scripts.activationMessage.path()
