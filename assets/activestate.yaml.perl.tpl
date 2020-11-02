scripts:
  - name: activationMessage
    language: perl
    value: |
        $out = <<EOT;
            You are now in an activated state, which is like a virtual environment to work
            in that doesn't affect the rest of your system. To leave, run `exit`.

            What's next?
            - To learn more about what you can do, run → `state --help`
            - To modify this runtime like adding packages or platforms, visit https://platform.activestate.com/{{.Owner}}/{{.Project}}
        EOT
        $out =~ s/^ +//gm;
        print $out;
events:
  # This is the ACTIVATE event, it will run whenever a new virtual environment is created (eg. by running `state activate`)
  # On Linux and macOS this will be ran as part of your shell's rc file, so you can use it to set up aliases, functions, environment variables, etc.
  - name: ACTIVATE
    value: {{.LangExe}} $scripts.activationMessage.path()
