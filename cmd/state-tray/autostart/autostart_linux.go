package autostart

func init() {
	Options.LaunchFileName = constants.TrayAppName
	Options.IconFileName = constants.TrayIconFileName
	Options.IconFileSource = constants.IconFileSource
	Options.GenericName = constants.TrayGenericName
	Options.Comment = constants.TrayComment
	Options.Keywords = constants.TrayKeywords
}
