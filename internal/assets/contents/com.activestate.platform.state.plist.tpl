<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Label</key>
        <string>com.activestate.StateToolService</string>
        <key>ProgramArguments</key>
        <array>
          <string>{{.Exec}}</string>
          {{- if .Args }}
          <string>{{.Args}}</string>
          {{- end}}
        </array>
        <key>RunAtLoad</key>
        <true/>
        <key>KeepAlive</key>
        <false/>
    </dict>
</plist>
