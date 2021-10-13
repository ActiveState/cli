scripts:
  - name: activationMessage
    language: perl
    value: |
        $out = <<EOT;
            Quick Start
            ───────────
            • To add a package to your runtime, type "state install <package name>"
            • Learn more about how to use the State Tool, type "state learn"
        EOT
        $out =~ s/^ +//gm;
        print $out;
events:
  # This is the ACTIVATE event, it will run whenever a new virtual environment is created (eg. by running `state activate`)
  # On Linux and macOS this will be ran as part of your shell's rc file, so you can use it to set up aliases, functions, environment variables, etc.
  - name: ACTIVATE
    value: {{.LangExe}} $scripts.activationMessage.path()
