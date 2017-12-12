Install dependencies:

```
go get -u github.com/spf13/cobra/cobra
```

Run CLI:

```
go run main.go install foo
```

Notes:

 - We need to be able to both download a precompiled distro as well as to pull in updates for that distro
   - essentially we will have to be able to also pull in a distro "package by package"
 - We'll need an API call that asks for what to download, with the request specifying whether this is a clean checkout or an update, and if so where are we updating from
   - OR we just always do a full download, but we shouldn't assume our users have unlimited bandwidth
   - Download packages based on preset URLs? eg. zeridian.io/download?package=foo&version=bar. This way the CLI tool can just go to town without requiring the API to spit out urls.