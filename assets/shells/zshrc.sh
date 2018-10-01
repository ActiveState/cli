if [ -f ~/.zshrc ]; then source ~/.zshrc; fi
export PROMPT_COMMAND="echo -e \"\033[1mActive state: {{.Project.Owner}}/{{.Project.Name}}\\033[0m\""
precmd() { eval "$PROMPT_COMMAND" }
{{range $K, $V := .Env}}
export {{$K}}="{{$V}}"
{{end}}
cd "{{.WD}}"

{{.UserScripts}}
