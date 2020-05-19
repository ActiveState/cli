package update

import (
	"errors"
	"testing"
)

func Test_run(t *testing.T) {
	type args struct {
		lock     bool
		isLocked bool
		force    bool
	}
	tests := []struct {
		name                      string
		args                      args
		confirmation              error
		wantErr                   bool
		wantRunLockCalled         bool
		wantRunUpdateLockCalled   bool
		wantConfirmLockCalled     bool
		wantRunUpdateGlobalCalled bool
	}{
		{
			"Updates global",
			args{
				false,
				false,
				false,
			},
			nil,
			false,
			false,
			false,
			false,
			true,
		},
		{
			"Locks",
			args{
				true,
				false,
				false,
			},
			nil,
			false,
			true,
			false,
			false,
			false,
		},
		{
			"Updates locked",
			args{
				false,
				true,
				false,
			},
			nil,
			false,
			false,
			true,
			true,
			false,
		},
		{
			"Fails to update locked due to confirmation",
			args{
				false,
				true,
				false,
			},
			errors.New("Cancel"),
			true,
			false,
			false,
			true,
			false,
		},
		{
			"Updates without confirmation",
			args{
				false,
				true,
				true,
			},
			nil,
			false,
			false,
			true,
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				runLockCalled         bool
				runUpdateLockCalled   bool
				confirmLockCalled     bool
				runUpdateGlobalCalled bool
			)
			err := run(
				tt.args.lock, tt.args.isLocked, tt.args.force,
				func() error { runLockCalled = true; return nil },
				func() error { runUpdateLockCalled = true; return nil },
				func() error { runUpdateGlobalCalled = true; return nil },
				func() error { confirmLockCalled = true; return tt.confirmation },
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if runLockCalled != tt.wantRunLockCalled {
				if tt.wantRunLockCalled {
					t.Errorf("Project should be locked but lock was not called, args: %v", tt.args)
				} else {
					t.Errorf("Project should not be locked but lock was called, args: %v", tt.args)
				}
			}
			if runUpdateLockCalled != tt.wantRunUpdateLockCalled {
				if tt.wantRunUpdateLockCalled {
					t.Errorf("Locked project should be updated but update for project was not called, args: %v", tt.args)
				} else {
					t.Errorf("Locked project should not be updated but update for project was called, args: %v", tt.args)
				}
			}
			if runUpdateGlobalCalled != tt.wantRunUpdateGlobalCalled {
				if tt.wantRunUpdateGlobalCalled {
					t.Errorf("Global state tool should be updated but update was not called, args: %v", tt.args)
				} else {
					t.Errorf("Global state tool should not be updated but update was called, args: %v", tt.args)
				}
			}
			if confirmLockCalled != tt.wantConfirmLockCalled {
				if tt.wantConfirmLockCalled {
					t.Errorf("Confirmation should have been asked but this did not occur, args: %v", tt.args)
				} else {
					t.Errorf("Confirmation should not have been asked but this did occur, args: %v", tt.args)
				}
			}
		})
	}
}
