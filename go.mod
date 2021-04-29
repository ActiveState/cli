module github.com/ActiveState/cli

go 1.16

replace github.com/ActiveState/cli => ./

require (
	cloud.google.com/go v0.64.0
	github.com/99designs/gqlgen v0.13.0
	github.com/ActiveState/archiver v3.1.1+incompatible
	github.com/ActiveState/go-ogle-analytics v0.0.0-20170510030904-9b3f14901527
	github.com/ActiveState/sysinfo v0.0.0-20200619170619-0582d42daf27
	github.com/ActiveState/termtest v0.7.1
	github.com/ActiveState/termtest/expect v0.7.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/PuerkitoBio/purell v1.1.0 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/agnivade/levenshtein v1.1.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/asaskevich/govalidator v0.0.0-20180315120708-ccb8e960c48f // indirect
	github.com/aws/aws-sdk-go v1.13.8
	github.com/blang/semver v3.5.1+incompatible
	github.com/creack/pty v1.1.11
	github.com/dave/jennifer v0.18.0
	github.com/denisbrodbeck/machineid v0.8.0
	github.com/dsnet/compress v0.0.0-20171208185109-cc9eb1d7ad76 // indirect
	github.com/faiface/mainthread v0.0.0-20171120011319-8b78f0a41ae3
	github.com/fatih/color v1.10.0
	github.com/felixge/fgprof v0.9.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gammazero/workerpool v1.1.1
	github.com/getlantern/systray v1.1.0
	github.com/go-ini/ini v1.32.0 // indirect
	github.com/go-ole/go-ole v1.2.4
	github.com/go-openapi/analysis v0.0.0-20180418034448-863ac7f90e00 // indirect
	github.com/go-openapi/errors v0.0.0-20171226161601-7bcb96a367ba
	github.com/go-openapi/jsonpointer v0.0.0-20180322222829-3a0015ad55fa // indirect
	github.com/go-openapi/jsonreference v0.0.0-20180322222742-3fb327e6747d // indirect
	github.com/go-openapi/loads v0.0.0-20171207192234-2a2b323bab96 // indirect
	github.com/go-openapi/runtime v0.0.0-20180420041453-f12926f16ac2
	github.com/go-openapi/spec v0.0.0-20180415031709-bcff419492ee // indirect
	github.com/go-openapi/strfmt v0.0.0-20180407011102-481808443b00
	github.com/go-openapi/swag v0.0.0-20180405201759-811b1089cde9
	github.com/go-openapi/validate v0.0.0-20180422194751-f8f9c5961cd5
	github.com/gobuffalo/packr v1.10.7
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/uuid v1.1.2
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-retryablehttp v0.6.7
	github.com/hashicorp/go-version v1.1.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/hpcloud/tail v1.0.0
	github.com/iafan/cwalk v0.0.0-20191125092548-dd7f505d2f66
	github.com/imdario/mergo v0.3.11
	github.com/jarcoal/httpmock v1.0.3
	github.com/jessevdk/go-flags v1.4.0
	github.com/jmespath/go-jmespath v0.0.0-20160202185014-0b12d6b521d8 // indirect
	github.com/kami-zh/go-capturer v0.0.0-20171211120116-e492ea43421d
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1
	github.com/labstack/echo/v4 v4.2.1
	github.com/machinebox/graphql v0.2.2
	github.com/mailru/easyjson v0.0.0-20180323154445-8b799c424f57 // indirect
	github.com/mash/go-tempfile-suffix v0.0.0-20150731093933-48f0f8a3a5ab
	github.com/matryer/is v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.10
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/nicksnyder/go-i18n v1.10.0
	github.com/nwaples/rardecode v0.0.0-20171029023500-e06696f847ae // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/phayes/permbits v0.0.0-20190108233746-1efae4548023
	github.com/pierrec/lz4 v0.0.0-20190222153722-062282ea0dcf // indirect
	github.com/pkg/errors v0.9.1
	github.com/posener/wstest v0.0.0-20180216222922-04b166ca0bf1
	github.com/rollbar/rollbar-go v1.1.0
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/shirou/gopsutil v2.19.12+incompatible
	github.com/skratchdot/open-golang v0.0.0-20190104022628-a2dfa6d0dab6
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.6.2-0.20201103103935-92707c0b2d50
	github.com/thoas/go-funk v0.8.0
	github.com/ulikunitz/xz v0.5.4 // indirect
	github.com/vbauerster/mpb/v6 v6.0.2
	github.com/vektah/gqlparser/v2 v2.1.0
	github.com/wailsapp/wails v1.16.3
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yuin/goldmark v1.1.32
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/net v0.0.0-20210420210106-798c2154c571
	golang.org/x/sys v0.0.0-20210420205809-ac73e9fd8988
	golang.org/x/text v0.3.6
	google.golang.org/genproto v0.0.0-20200815001618-f69a88009b70
	google.golang.org/grpc v1.36.0 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/mgo.v2 v2.0.0-20160818020120-3f83fa500528 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
