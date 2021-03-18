package request

type Request interface {
	Query() string
	Vars() map[string]interface{}
}

type Client interface {
	Run(request Request, response interface{}) error
}
