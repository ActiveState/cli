// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graph

import (
	"fmt"
	"io"
	"strconv"
)

type AnalyticsEventResponse struct {
	Sent bool `json:"sent"`
}

type AvailableUpdate struct {
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Path     string `json:"path"`
	Platform string `json:"platform"`
	Sha256   string `json:"sha256"`
}

type ConfigChangedResponse struct {
	Received bool `json:"received"`
}

type GlobFileResult struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Hash    string `json:"hash"`
}

type GlobResult struct {
	Files []*GlobFileResult `json:"files"`
	Hash  string            `json:"hash"`
}

type Jwt struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type MessageInfo struct {
	ID        string               `json:"id"`
	Message   string               `json:"message"`
	Condition string               `json:"condition"`
	Repeat    MessageRepeatType    `json:"repeat"`
	Interrupt MessageInterruptType `json:"interrupt"`
	Placement MessagePlacementType `json:"placement"`
}

type Mutation struct {
}

type Organization struct {
	URLname string `json:"URLname"`
	Role    string `json:"role"`
}

type ProcessInfo struct {
	Exe string `json:"exe"`
	Pid int    `json:"pid"`
}

type Project struct {
	Namespace string   `json:"namespace"`
	Locations []string `json:"locations"`
}

type Query struct {
}

type ReportRuntimeUsageResponse struct {
	Received bool `json:"received"`
}

type StateVersion struct {
	License  string `json:"license"`
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Revision string `json:"revision"`
	Date     string `json:"date"`
}

type User struct {
	UserID        string          `json:"userID"`
	Username      string          `json:"username"`
	Email         string          `json:"email"`
	Organizations []*Organization `json:"organizations"`
}

type Version struct {
	State *StateVersion `json:"state"`
}

type MessageInterruptType string

const (
	MessageInterruptTypeDisabled MessageInterruptType = "Disabled"
	MessageInterruptTypePrompt   MessageInterruptType = "Prompt"
	MessageInterruptTypeExit     MessageInterruptType = "Exit"
)

var AllMessageInterruptType = []MessageInterruptType{
	MessageInterruptTypeDisabled,
	MessageInterruptTypePrompt,
	MessageInterruptTypeExit,
}

func (e MessageInterruptType) IsValid() bool {
	switch e {
	case MessageInterruptTypeDisabled, MessageInterruptTypePrompt, MessageInterruptTypeExit:
		return true
	}
	return false
}

func (e MessageInterruptType) String() string {
	return string(e)
}

func (e *MessageInterruptType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = MessageInterruptType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid MessageInterruptType", str)
	}
	return nil
}

func (e MessageInterruptType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type MessagePlacementType string

const (
	MessagePlacementTypeBeforeCmd MessagePlacementType = "BeforeCmd"
	MessagePlacementTypeAfterCmd  MessagePlacementType = "AfterCmd"
)

var AllMessagePlacementType = []MessagePlacementType{
	MessagePlacementTypeBeforeCmd,
	MessagePlacementTypeAfterCmd,
}

func (e MessagePlacementType) IsValid() bool {
	switch e {
	case MessagePlacementTypeBeforeCmd, MessagePlacementTypeAfterCmd:
		return true
	}
	return false
}

func (e MessagePlacementType) String() string {
	return string(e)
}

func (e *MessagePlacementType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = MessagePlacementType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid MessagePlacementType", str)
	}
	return nil
}

func (e MessagePlacementType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type MessageRepeatType string

const (
	MessageRepeatTypeDisabled   MessageRepeatType = "Disabled"
	MessageRepeatTypeConstantly MessageRepeatType = "Constantly"
	MessageRepeatTypeHourly     MessageRepeatType = "Hourly"
	MessageRepeatTypeDaily      MessageRepeatType = "Daily"
	MessageRepeatTypeWeekly     MessageRepeatType = "Weekly"
	MessageRepeatTypeMonthly    MessageRepeatType = "Monthly"
)

var AllMessageRepeatType = []MessageRepeatType{
	MessageRepeatTypeDisabled,
	MessageRepeatTypeConstantly,
	MessageRepeatTypeHourly,
	MessageRepeatTypeDaily,
	MessageRepeatTypeWeekly,
	MessageRepeatTypeMonthly,
}

func (e MessageRepeatType) IsValid() bool {
	switch e {
	case MessageRepeatTypeDisabled, MessageRepeatTypeConstantly, MessageRepeatTypeHourly, MessageRepeatTypeDaily, MessageRepeatTypeWeekly, MessageRepeatTypeMonthly:
		return true
	}
	return false
}

func (e MessageRepeatType) String() string {
	return string(e)
}

func (e *MessageRepeatType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = MessageRepeatType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid MessageRepeatType", str)
	}
	return nil
}

func (e MessageRepeatType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
