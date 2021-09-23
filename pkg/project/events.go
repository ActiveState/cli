package project

type EventType string

const (
	BeforeCmd     EventType = "before-command"
	AfterCmd      EventType = "after-command"
	Activate      EventType = "activate"
	FirstActivate EventType = "first-activate"
)

func (e EventType) String() string {
	return string(e)
}

func ActivateEvents() []EventType {
	return []EventType{
		Activate,
		FirstActivate,
	}
}
