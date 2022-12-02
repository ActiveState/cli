<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Label</key>
        <string>{{.Label}}</string>
        <key>ProgramArguments</key>
        <array>
          <string>{{.Exec}}</string>
          {{- if .Args }}
          <string>{{.Args}}</string>
          {{- end}}
        </array>
        {{- if .Interactive }}
        <key>ProcessType</key>
        <string>Interactive</string>
        {{- end}}
        <key>RunAtLoad</key>
        <true/>
        <key>KeepAlive</key>
        <false/>
    </dict>
</plist>
