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
  - name: ppm
    language: perl
    value: |
        use strict;
        use warnings;

        $ENV{ACTIVESTATE_SHIM} = 'ppm';

        mapcmds((
            "install" => "install",
            "search"  => "search",
            "upgrade" => "install",
            "remove"  => "uninstall",
            "list"    => "packages",
        ));

        print("Could not shim your command as it is not supported by the State Tool.\nPlease check 'state --help' to find " .
            "the best analog for the command you're trying to run.\n" .
            "To configure this shim edit the following file:\n${project.path()}/activestate.yaml\n");

        sub mapcmds {
            my (%entries) = @_;
            while ((my $from, my $to) = each(%entries)) {
                if ($ARGV[0] eq $from) {
                    printf("Shimming command to 'state %s'. To configure this shim, edit the following file:\n${project.path()}/activestate.yaml\n\n", $to);
                    system("state", $to, @ARGV[1 .. $#ARGV]);
                    exit($?);
                }
            }
        }
events:
  # This is the ACTIVATE event, it will run whenever a new virtual environment is created (eg. by running `state activate`)
  # On Linux and macOS this will be ran as part of your shell's rc file, so you can use it to set up aliases, functions, environment variables, etc.
  - name: ACTIVATE
    value: {{.LangExe}} $scripts.activationMessage.path()
