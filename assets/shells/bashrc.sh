if [ -f ~/.bashrc ]; then source ~/.bashrc; fi
export PROMPT_COMMAND="echo -e \"\033[1mActive state: {{.Project.Owner}}/{{.Project.Name}}\\033[0m\"
"
{{range $K, $V := .Env}}
export {{$K}}="{{$V}}"
{{end}}
cd "{{.WD}}"

{{.UserScripts}}