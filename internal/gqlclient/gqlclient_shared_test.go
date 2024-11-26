package gqlclient

type TestRequest struct {
	q       string
	vars    map[string]interface{}
	headers map[string][]string
}

func NewTestRequest(q string) *TestRequest {
	return &TestRequest{
		q:       q,
		vars:    make(map[string]interface{}),
		headers: make(map[string][]string),
	}
}

func (r *TestRequest) Query() string {
	return r.q
}

func (r *TestRequest) Vars() (map[string]interface{}, error) {
	return r.vars, nil
}

func (r *TestRequest) Headers() map[string][]string {
	return r.headers
}

type TestRequestWithFiles struct {
	*TestRequest
	files []File
}

func NewTestRequestWithFiles(q string) *TestRequestWithFiles {
	return &TestRequestWithFiles{
		TestRequest: NewTestRequest(q),
		files:       make([]File, 0),
	}
}

func (r *TestRequestWithFiles) Files() []File {
	return r.files
}
