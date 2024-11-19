package messages

import (
	"reflect"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/blang/semver"
)

func Test_check(t *testing.T) {
	baseTime := time.Now()
	type args struct {
		params        *ConditionParams
		messages      []*graph.MessageInfo
		lastReportMap map[string]interface{}
		baseTime      time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantIDs []string
		wantErr bool
	}{
		{
			"No special conditions",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A"}, {ID: "B"}, {ID: "C"},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A", "B", "C"},
			false,
		},
		{
			"Simple Command Condition",
			args{
				params: &ConditionParams{
					Command: "foo",
				},
				messages: []*graph.MessageInfo{
					{ID: "A", Condition: `eq .Command "bar"`},
					{ID: "B", Condition: `eq .Command "foo"`},
					{ID: "C", Condition: `eq .Command "foobar"`},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"B"},
			false,
		},
		{
			"'contains' Condition",
			args{
				params: &ConditionParams{
					UserEmail: "john@doe.org",
				},
				messages: []*graph.MessageInfo{
					{ID: "A", Condition: `contains .UserEmail "john"`},
					{ID: "B", Condition: `contains .UserEmail "fred"`},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A"},
			false,
		},
		{
			"String 'hasPrefix' Condition",
			args{
				params: &ConditionParams{
					UserEmail: "john@doe.org",
				},
				messages: []*graph.MessageInfo{
					{ID: "A", Condition: `hasPrefix .UserEmail "john"`},
					{ID: "B", Condition: `hasPrefix .UserEmail "org"`},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A"},
			false,
		},
		{
			"String 'hasSuffix' Condition",
			args{
				params: &ConditionParams{
					UserEmail: "john@doe.org",
				},
				messages: []*graph.MessageInfo{
					{ID: "A", Condition: `hasSuffix .UserEmail "john"`},
					{ID: "B", Condition: `hasSuffix .UserEmail "org"`},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"B"},
			false,
		},
		{
			"`regexMatch` Condition",
			args{
				params: &ConditionParams{
					UserEmail: "john@doe.org",
				},
				messages: []*graph.MessageInfo{
					{ID: "A", Condition: `regexMatch .UserEmail ".*@doe.org$"`},
					{ID: "B", Condition: `regexMatch .UserEmail "^doe.org$"`},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A"},
			false,
		},
		{
			"`regexMatch` Compilation Error",
			args{
				params: &ConditionParams{
					UserEmail: "john@doe.org",
				},
				messages: []*graph.MessageInfo{
					{ID: "A", Condition: `regexMatch .UserEmail ".*@doe.org$"`},
					{ID: "B", Condition: `regexMatch .UserEmail ".*("`},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A"},
			false,
		},
		{
			"Version Condition",
			args{
				params: &ConditionParams{
					StateVersion: NewVersionFromSemver(semver.MustParse("7.8.9-SHA123456a7b")),
				},
				messages: []*graph.MessageInfo{
					{ID: "A", Condition: `eq .StateVersion.Major 7`},
					{ID: "B", Condition: `eq .StateVersion.Minor 8`},
					{ID: "C", Condition: `eq .StateVersion.Patch 9`},
					{ID: "D", Condition: `hasSuffix .StateVersion.Raw "SHA123456a7b"`},
					{ID: "E", Condition: `eq .StateVersion.Major 1`},
					{ID: "F", Condition: `eq .StateVersion.Minor 2`},
					{ID: "G", Condition: `eq .StateVersion.Patch 3`},
					{ID: "H", Condition: `eq .StateVersion.Build "foo"`},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A", "B", "C", "D"},
			false,
		},
		{
			"Repeat Disabled",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", Repeat: graph.MessageRepeatTypeDisabled},
					{ID: "B", Repeat: graph.MessageRepeatTypeDisabled},
					{ID: "C", Repeat: graph.MessageRepeatTypeDisabled},
				},
				lastReportMap: map[string]interface{}{
					"A": baseTime.Format(time.RFC3339),
					"C": baseTime.Format(time.RFC3339),
				},
				baseTime: baseTime,
			},
			[]string{"B"},
			false,
		},
		{
			"Repeat Constantly",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", Repeat: graph.MessageRepeatTypeConstantly},
					{ID: "B", Repeat: graph.MessageRepeatTypeConstantly},
					{ID: "C", Repeat: graph.MessageRepeatTypeConstantly},
				},
				lastReportMap: map[string]interface{}{
					"A": baseTime.Format(time.RFC3339),
					"C": baseTime.Add(-time.Hour * 24 * 30).Format(time.RFC3339),
				},
				baseTime: baseTime,
			},
			[]string{"A", "B", "C"},
			false,
		},
		{
			"Repeat Hourly",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "B", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "C", Repeat: graph.MessageRepeatTypeHourly},
				},
				lastReportMap: map[string]interface{}{
					"A": baseTime.Format(time.RFC3339),
					"B": baseTime.Add(-time.Hour).Format(time.RFC3339),
					"C": baseTime.Format(time.RFC3339),
				},
				baseTime: baseTime,
			},
			[]string{"B"},
			false,
		},
		{
			"Repeat Daily",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "B", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "C", Repeat: graph.MessageRepeatTypeHourly},
				},
				lastReportMap: map[string]interface{}{
					"A": baseTime.Format(time.RFC3339),
					"B": baseTime.Add(-time.Hour * 24).Format(time.RFC3339),
					"C": baseTime.Format(time.RFC3339),
				},
				baseTime: baseTime,
			},
			[]string{"B"},
			false,
		},
		{
			"Repeat Weekly",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "B", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "C", Repeat: graph.MessageRepeatTypeHourly},
				},
				lastReportMap: map[string]interface{}{
					"A": baseTime.Format(time.RFC3339),
					"B": baseTime.Add(-time.Hour * 24 * 7).Format(time.RFC3339),
					"C": baseTime.Format(time.RFC3339),
				},
				baseTime: baseTime,
			},
			[]string{"B"},
			false,
		},
		{
			"Repeat Monthly",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "B", Repeat: graph.MessageRepeatTypeHourly},
					{ID: "C", Repeat: graph.MessageRepeatTypeHourly},
				},
				lastReportMap: map[string]interface{}{
					"A": baseTime.Format(time.RFC3339),
					"B": baseTime.Add(-time.Hour * 24 * 7 * 30).Format(time.RFC3339),
					"C": baseTime.Format(time.RFC3339),
				},
				baseTime: baseTime,
			},
			[]string{"B"},
			false,
		},
		{
			"Date Range - Within Range",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", StartDate: baseTime.Add(-24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(24 * time.Hour).Format(time.RFC3339)},
					{ID: "B", StartDate: baseTime.Add(-1 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(1 * time.Hour).Format(time.RFC3339)},
					{ID: "C", StartDate: baseTime.Add(1 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(24 * time.Hour).Format(time.RFC3339)},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A", "B"},
			false,
		},
		{
			"Date Range - No Dates Specified",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A"},
					{ID: "B", StartDate: baseTime.Add(-1 * time.Hour).Format(time.RFC3339)},
					{ID: "C", EndDate: baseTime.Add(1 * time.Hour).Format(time.RFC3339)},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A", "B", "C"},
			false,
		},
		{
			"Date Range - Invalid Date Format",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", StartDate: "invalid-date"},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{},
			true,
		},
		{
			"Date Range - Only Start Date",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", StartDate: baseTime.Add(-1 * time.Hour).Format(time.RFC3339)},
					{ID: "B", StartDate: baseTime.Add(1 * time.Hour).Format(time.RFC3339)},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A"},
			false,
		},
		{
			"Date Range - Only End Date",
			args{
				params: &ConditionParams{},
				messages: []*graph.MessageInfo{
					{ID: "A", EndDate: baseTime.Add(1 * time.Hour).Format(time.RFC3339)},
					{ID: "B", EndDate: baseTime.Add(-1 * time.Hour).Format(time.RFC3339)},
				},
				lastReportMap: map[string]interface{}{},
				baseTime:      baseTime,
			},
			[]string{"A"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := check(tt.args.params, tt.args.messages, tt.args.lastReportMap, tt.args.baseTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("check() error = %v, wantErr %v", errs.JoinMessage(err), tt.wantErr)
				return
			}
			gotIDs := []string{}
			for _, msg := range got {
				gotIDs = append(gotIDs, msg.ID)
			}
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Errorf("check() got = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}
