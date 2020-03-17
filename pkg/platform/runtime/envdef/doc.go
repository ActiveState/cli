// Package envdef implements a parser for the runtime environment for alternative builds
//
// Builds that are built with the alternative build environment, include
// runtime.json files that define which environment variables need to be set to
// install and use the provided artifacts.
// The schema of this file can be downloaded [here](https://drive.google.com/drive/u/0/my-drive)
//
// The same parser and interpreter also exists in [TheHomeRepot](https://github.com/ActiveState/TheHomeRepot/blob/master/service/build-wrapper/wrapper/runtime.py)
//
// Changes to the runtime environment definition schema should be synchronized
// between these two places. For now, this can be most easily accomplished by
// keeping the description of test cases in the [cli repo](https://github.com/ActiveState/cli/blob/master/pkg/platform/runtime/envdef/runtime_test_cases.json)
// and [TheHomeRepot](https://github.com/ActiveState/TheHomeRepot/blob/master/service/build-wrapper/runtime_test_cases.json)
// in sync.
package envdef
