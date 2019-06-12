if [ -f ~/.bashrc ]; then source ~/.bashrc; fi

if [ -z "$PROMPT_COMMAND" ]; then
    export PROMPT_COMMAND="echo -e '\033[1mActive state: {{.Owner}}/{{.Name}}\\033[0m'"
fi

{{range $K, $V := .Env}}
export {{$K}}="{{$V}}"
{{end}}

{{range $K, $CMD := .Scripts}}
alias {{$K}}='state run {{$CMD}}'
{{end}}

cd "{{.WD}}"

{{.UserScripts}}