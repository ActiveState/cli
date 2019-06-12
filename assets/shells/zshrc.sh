if [ -f $ZDOTDIR/.zshrc ]; then source $ZDOTDIR/.zshrc; fi
export PROMPT_COMMAND="echo -e \"\033[1mActive state: {{.Owner}}/{{.Name}}\\033[0m\"
"
precmd() { eval "$PROMPT_COMMAND" }
{{range $K, $V := .Env}}
export {{$K}}="{{$V}}"
{{end}}

{{range $K, $CMD := .Scripts}}
alias {{$K}}='state run {{$CMD}}'
{{end}}

cd "{{.WD}}"

{{.UserScripts}}
