package request

type JWTRequest struct {
}

func NewJWTRequest() *JWTRequest {
	return &JWTRequest{}
}

func (m *JWTRequest) Query() string {
	return `query()  {
		getJWT() {
			token
			user {
				userID
				username
				email
				organizations {
					URLname
					role
				}
			}
		}
	}`
}

func (m *JWTRequest) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
