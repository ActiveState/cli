package autostart

var data = map[AppName]options{
	Tray: {
		launchFileName: "com.activestate.platform.state-tray.plist",
	},
	Service: {
		launchFileName: "com.activestate.platform.state-svc.plist",
	},
}
