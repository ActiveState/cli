package wc

import "testing"

func Test_sprint(t *testing.T) {
	type args struct {
		depth int
		msg   string
		args  []interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Hello world depth 0",
			args: args{
				depth: 0,
				msg:   "Hello world",
			},
			want: "Hello world",
		},
		{
			name: "Hello world depth 1",
			args: args{
				depth: 1,
				msg:   "Hello world",
			},
			want: "  |- Hello world",
		},
		{
			name: "Hello world depth 1 multi line",
			args: args{
				depth: 1,
				msg:   "Hello \nworld\n!",
			},
			want: "  |- Hello \n     world\n     !",
		},
		{
			name: "Hello world depth 1 multi line in args",
			args: args{
				depth: 1,
				msg:   "%s",
				args:  []interface{}{"Hello \nworld\n!"},
			},
			want: "  |- Hello \n     world\n     !",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sprint(tt.args.depth, tt.args.msg, tt.args.args...); got != tt.want {
				t.Errorf("sprint() = '%v', want '%v'", got, tt.want)
			}
		})
	}
}
