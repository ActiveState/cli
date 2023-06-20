package rollbar

import "testing"

func Test_doNotReport_Contains(t *testing.T) {
	type args struct {
		msg   string
		toAdd []string
	}
	tests := []struct {
		name string
		d    doNotReport
		args args
		want bool
	}{
		{
			name: "empty",
			d:    doNotReport{},
			args: args{
				msg: "test",
				toAdd: []string{
					"",
				},
			},
			want: false,
		},
		{
			name: "whitespace",
			d:    doNotReport{},
			args: args{
				msg: "test",
				toAdd: []string{
					" ",
				},
			},
			want: false,
		},
		{
			name: "contains",
			d:    doNotReport{},
			args: args{
				msg: "test",
				toAdd: []string{
					"test",
				},
			},
			want: true,
		},
		{
			name: "not contains",
			d:    doNotReport{},
			args: args{
				msg: "test",
				toAdd: []string{
					"test1",
				},
			},
			want: false,
		},
		{
			name: "multiple",
			d:    doNotReport{},
			args: args{
				msg: "test",
				toAdd: []string{
					"test1",
					"test2",
					"test3",
				},
			},
			want: false,
		},
		{
			name: "multiple contains",
			d:    doNotReport{},
			args: args{
				msg: "test",
				toAdd: []string{
					"test1",
					"test2",
					"test3",
					"test",
				},
			},
			want: true,
		},
		{
			name: "complex message not contains",
			d:    doNotReport{},
			args: args{
				msg: "This is a complex message",
				toAdd: []string{
					"A test message",
					"A complex message",
				},
			},
			want: false,
		},
		{
			name: "complex message contains",
			d:    doNotReport{},
			args: args{
				msg: "This is a complex message",
				toAdd: []string{
					"A test message",
					"A complex message",
					"This is a complex message",
				},
			},
			want: true,
		},
		{
			name: "complex message contains case insensitive",
			d:    doNotReport{},
			args: args{
				msg: "This is a complex message",
				toAdd: []string{
					"A test message",
					"A complex message",
					"this is a complex message",
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, msg := range tt.args.toAdd {
				tt.d.Add(msg)
			}
			if got := tt.d.Contains(tt.args.msg); got != tt.want {
				t.Errorf("doNotReport.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
