# {{.Start}}

{{range $K, $V := .Env}}
export {{$K}}="{{$V}}"
{{end}}

# {{.Stop}}