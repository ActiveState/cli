module github.com/ActiveState/cli/scripts/error-reports

go 1.16

replace github.com/davidji99/rollrest-go => ./rollbar-client

require (
	github.com/adrg/strutil v0.2.3
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/davidji99/rollrest-go v0.0.0-00010101000000-000000000000
	github.com/gizak/termui/v3 v3.1.0
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/sys v0.0.0-20211105183446-c75c47738b0c // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
