#!/usr/bin/env bash



echo  "STABLE_BRANCHNAME "  `git rev-parse --abbrev-ref HEAD`

# TODO:
# convert this
echo "STABLE_BUILDNUMBER SETME!"
    # Constants["BuildNumber"] = func() string {
    #     out := getCmdOutput("git rev-list --abbrev-commit HEAD")
    #     return strconv.Itoa(len(strings.Split(out, "\n")))
    # }


#     # Constants["RevisionHash"] = func() string { return getCmdOutput("git rev-parse --verify HEAD") }
echo "STABLE_REVISIONHASH" `git rev-parse --verify HEAD`

# TODO: Set me when BUILDNUMBER is corrected
#     # Constants["Version"] = func() string { return fmt.Sprintf("%s-%s", constants.VersionNumber, Constants["BuildNumber"]()) }

