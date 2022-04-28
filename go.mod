module github.com/ActiveState/cli

go 1.16

replace github.com/ActiveState/cli => ./

require (
	cloud.google.com/go v0.64.0
	github.com/99designs/gqlgen v0.13.0
	github.com/ActiveState/archiver v3.1.1+incompatible
	github.com/ActiveState/go-ogle-analytics v0.0.0-20170510030904-9b3f14901527
	github.com/ActiveState/termtest v0.7.1
	github.com/ActiveState/termtest/expect v0.7.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/agnivade/levenshtein v1.1.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/aws/aws-sdk-go v1.34.28
	github.com/blang/semver v3.5.1+incompatible
	github.com/creack/pty v1.1.11
	github.com/dave/jennifer v0.18.0
	github.com/dsnet/compress v0.0.0-20171208185109-cc9eb1d7ad76 // indirect
	github.com/faiface/mainthread v0.0.0-20171120011319-8b78f0a41ae3
	github.com/fatih/color v1.10.0
	github.com/felixge/fgprof v0.9.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gammazero/workerpool v1.1.1
	github.com/getlantern/systray v1.1.0
	github.com/go-ole/go-ole v1.2.6
	github.com/go-openapi/errors v0.20.0
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/runtime v0.19.29
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.2
	github.com/gofrs/flock v0.8.1
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/uuid v1.1.2
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-retryablehttp v0.6.7
	github.com/hashicorp/go-version v1.1.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hpcloud/tail v1.0.0
	github.com/imdario/mergo v0.3.11
	github.com/jarcoal/httpmock v1.0.3
	github.com/jessevdk/go-flags v1.4.0
	github.com/kami-zh/go-capturer v0.0.0-20171211120116-e492ea43421d
	github.com/labstack/echo/v4 v4.2.1
	github.com/machinebox/graphql v0.2.2
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mash/go-tempfile-suffix v0.0.0-20150731093933-48f0f8a3a5ab
	github.com/matryer/is v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mattn/go-runewidth v0.0.13
	github.com/mattn/go-sqlite3 v1.14.7 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nicksnyder/go-i18n v1.10.0
	github.com/nwaples/rardecode v0.0.0-20171029023500-e06696f847ae // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/phayes/permbits v0.0.0-20190108233746-1efae4548023
	github.com/pierrec/lz4 v0.0.0-20190222153722-062282ea0dcf // indirect
	github.com/posener/wstest v0.0.0-20180216222922-04b166ca0bf1
	github.com/rollbar/rollbar-go v1.1.0
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/skratchdot/open-golang v0.0.0-20190104022628-a2dfa6d0dab6
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.8.0
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/ulikunitz/xz v0.5.4 // indirect
	github.com/vbauerster/mpb/v7 v7.1.5
	github.com/vektah/gqlparser/v2 v2.1.0
	github.com/wailsapp/wails v1.16.3
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yuin/goldmark v1.3.5
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.mongodb.org/mongo-driver v1.5.3 // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d
	golang.org/x/text v0.3.6
	google.golang.org/genproto v0.0.0-20200815001618-f69a88009b70
	google.golang.org/grpc v1.36.0 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0
	modernc.org/sqlite v1.11.2
)
