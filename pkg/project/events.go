package project

type EventType string

const (
	BeforeCmd EventType = "before-command"
	AfterCmd  EventType = "after-command"
)
