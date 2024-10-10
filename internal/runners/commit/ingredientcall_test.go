package commit

import "testing"

func Test_hashFuncCall(t *testing.T) {
	type args struct {
		fc   *buildscript.FuncCall
		seed string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hashFuncCall(tt.args.fc, tt.args.seed)
			if (err != nil) != tt.wantErr {
				t.Errorf("hashFuncCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("hashFuncCall() got = %v, want %v", got, tt.want)
			}
		})
	}
}
