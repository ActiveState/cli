source ~/.bashrc
export PROMPT_COMMAND="echo -e \"\033[1mExecuted under state: {{.Owner}}/{{.Name}}\\033[0m\""