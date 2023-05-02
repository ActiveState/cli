module github.com/ActiveState/cli

go 1.20

replace cloud.google.com/go => cloud.google.com/go v0.110.0

require (
	cloud.google.com/go/compute/metadata v0.2.3
	cloud.google.com/go/secretmanager v1.9.0
	github.com/99designs/gqlgen v0.17.19
	github.com/ActiveState/go-ogle-analytics v0.0.0-20170510030904-9b3f14901527
	github.com/ActiveState/termtest v0.7.2
	github.com/ActiveState/termtest/expect v0.7.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/andygrunwald/go-jira v1.15.1
	github.com/aws/aws-sdk-go v1.34.28
	github.com/blang/semver v3.5.1+incompatible
	github.com/creack/pty v1.1.11
	github.com/dave/jennifer v0.18.0
	github.com/faiface/mainthread v0.0.0-20171120011319-8b78f0a41ae3
	github.com/fatih/color v1.10.0
	github.com/felixge/fgprof v0.9.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gammazero/workerpool v1.1.1
	github.com/go-ole/go-ole v1.2.6
	github.com/go-openapi/errors v0.20.0
	github.com/go-openapi/runtime v0.19.29
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.2
	github.com/gofrs/flock v0.8.1
	github.com/google/go-github/v45 v45.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-retryablehttp v0.6.7
	github.com/hashicorp/go-version v1.1.0
	github.com/hpcloud/tail v1.0.0
	github.com/imdario/mergo v0.3.11
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/jarcoal/httpmock v1.0.3
	github.com/jessevdk/go-flags v1.4.0
	github.com/kami-zh/go-capturer v0.0.0-20171211120116-e492ea43421d
	github.com/labstack/echo/v4 v4.9.0
	github.com/machinebox/graphql v0.2.2
	github.com/mash/go-tempfile-suffix v0.0.0-20150731093933-48f0f8a3a5ab
	github.com/mattn/go-runewidth v0.0.13
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nicksnyder/go-i18n v1.10.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/phayes/permbits v0.0.0-20190108233746-1efae4548023
	github.com/posener/wstest v0.0.0-20180216222922-04b166ca0bf1
	github.com/rollbar/rollbar-go v1.1.0
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/shirou/gopsutil/v3 v3.22.7
	github.com/skratchdot/open-golang v0.0.0-20190104022628-a2dfa6d0dab6
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	github.com/thoas/go-funk v0.8.0
	github.com/vbauerster/mpb/v7 v7.1.5
	github.com/vektah/gqlparser/v2 v2.5.1
	go.mozilla.org/pkcs7 v0.0.0-20210826202110-33d05740a352
	golang.org/x/crypto v0.7.0
	golang.org/x/net v0.8.0
	golang.org/x/sys v0.6.0
	golang.org/x/term v0.6.0
	golang.org/x/text v0.8.0
	google.golang.org/genproto v0.0.0-20230209215440-0dfe4f8abfcc
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/toast.v1 v1.0.0-20180812000517-0a84660828b2
	gopkg.in/yaml.v2 v2.4.0
	modernc.org/sqlite v1.11.2
)

require (
	cloud.google.com/go/compute v1.18.0 // indirect
	cloud.google.com/go/iam v0.8.0 // indirect
	github.com/ActiveState/termtest/conpty v0.5.0 // indirect
	github.com/ActiveState/termtest/xpty v0.6.0 // indirect
	github.com/ActiveState/vt10x v1.3.1 // indirect
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/Netflix/go-expect v0.0.0-20201125194554-85d881c3777e // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/frankban/quicktest v1.14.4 // indirect
	github.com/gammazero/deque v0.0.0-20200721202602-07291166fe33 // indirect
	github.com/go-openapi/analysis v0.20.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/loads v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang-jwt/jwt/v4 v4.3.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/pprof v0.0.0-20200708004538-1a94d8640e99 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.7.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20190725054713-01f96b0aa0cd // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/labstack/gommon v0.3.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matryer/is v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.7 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/nwaples/rardecode v1.1.3 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	github.com/xanzy/ssh-agent v0.2.1 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.mongodb.org/mongo-driver v1.5.3 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/oauth2 v0.5.0 // indirect
	golang.org/x/time v0.1.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/api v0.110.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/grpc v1.53.0 // indirect
	google.golang.org/protobuf v1.29.0 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.2 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1
	howett.net/plist v1.0.0
	lukechampine.com/uint128 v1.1.1 // indirect
	modernc.org/cc/v3 v3.33.6 // indirect
	modernc.org/ccgo/v3 v3.9.5 // indirect
	modernc.org/libc v1.9.11 // indirect
	modernc.org/mathutil v1.4.0 // indirect
	modernc.org/memory v1.0.4 // indirect
	modernc.org/opt v0.1.1 // indirect
	modernc.org/strutil v1.1.1 // indirect
	modernc.org/token v1.0.0 // indirect
)
