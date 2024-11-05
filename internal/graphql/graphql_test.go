package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_DataPath(t *testing.T) {
	mockResponse := map[string]interface{}{
		"data": map[string]interface{}{
			"users": map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "1", "name": "Alice"},
					{"id": "2", "name": "Bob"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	tests := []struct {
		name     string
		query    string
		dataPath string
		want     interface{}
	}{
		{
			name: "no data path",
			query: `{
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "",
			want: map[string]interface{}{
				"users": map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{"id": "1", "name": "Alice"},
						map[string]interface{}{"id": "2", "name": "Bob"},
					},
				},
			},
		},
		{
			name: "with data path to users",
			query: `{
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "users",
			want: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		},
		{
			name: "named query without data path",
			query: `query GetUsers {
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "",
			want: map[string]interface{}{
				"users": map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{"id": "1", "name": "Alice"},
						map[string]interface{}{"id": "2", "name": "Bob"},
					},
				},
			},
		},
		{
			name: "named query with data path",
			query: `query GetUsers {
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "users",
			want: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		},
		{
			name: "named mutation",
			query: `mutation CreateUser {
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "users",
			want: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		},
		{
			name: "anonymous query with items data path",
			query: `{
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "items",
			want: []interface{}{
				map[string]interface{}{"id": "1", "name": "Alice"},
				map[string]interface{}{"id": "2", "name": "Bob"},
			},
		},
		{
			name: "named query with items data path",
			query: `query GetUsers {
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "items",
			want: []interface{}{
				map[string]interface{}{"id": "1", "name": "Alice"},
				map[string]interface{}{"id": "2", "name": "Bob"},
			},
		},
		{
			name: "named mutation with items data path",
			query: `mutation CreateUser {
				users {
					items {
						id
						name
					}
				}
			}`,
			dataPath: "items",
			want: []interface{}{
				map[string]interface{}{"id": "1", "name": "Alice"},
				map[string]interface{}{"id": "2", "name": "Bob"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewRequest(tt.query)
			req.DataPath(tt.dataPath)

			var resp interface{}
			err := client.Run(context.Background(), req, &resp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal response: %v", err)
			}

			want, err := json.Marshal(tt.want)
			if err != nil {
				t.Fatalf("failed to marshal expected result: %v", err)
			}

			if string(got) != string(want) {
				t.Errorf("got %s, want %s", got, want)
			}
		})
	}
}

func Test_inferDataPath(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name: "simple query",
			query: `{
				users {
					id
					name
				}
			}`,
			want: "users",
		},
		{
			name: "query with variables",
			query: `query ($id: ID!) {
				user(id: $id) {
					name
					email
				}
			}`,
			want: "user",
		},
		{
			name: "query with operation name",
			query: `query GetUser {
				user {
					name
				}
			}`,
			want: "user",
		},
		{
			name: "query with multiple fields",
			query: `{
				user {
					name
				}
				posts {
					title
				}
			}`,
			want: "user",
		},
		{
			name:  "empty query",
			query: "",
			want:  "",
		},
		{
			name: "alternate empty query",
			query: `query {
				query {}
			}`,
			want: "query",
		},
		{
			name: "malformed query",
			query: `query {
				badly formatted*
			}`,
			want: "badly",
		},
		{
			name: "named mutation",
			query: `mutation CreateUser($input: UserInput!) {
				createUser(input: $input) {
					id
					name
				}
			}`,
			want: "createUser",
		},
		{
			name: "named query with aliases",
			query: `query GetUserDetails {
				userInfo: user {
					profile {
						firstName
						lastName
					}
					settings {
						theme
					}
				}
			}`,
			want: "userInfo",
		},
		{
			name: "complex nested query",
			query: `query FetchDashboard {
				dashboard {
					widgets {
						id
						data {
							chart {
								points
							}
						}
					}
				}
			}`,
			want: "dashboard",
		},
		{
			name: "query with fragments",
			query: `query GetUserWithPosts {
				user {
					...UserFields
					posts {
						...PostFields
					}
				}
			}
			
			fragment UserFields on User {
				id
				name
			}
			
			fragment PostFields on Post {
				title
				content
			}`,
			want: "user",
		},
		{
			name: "subscription",
			query: `subscription OnUserUpdate {
				userUpdated {
					id
					status
				}
			}`,
			want: "userUpdated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferDataPath(tt.query); got != tt.want {
				t.Errorf("inferDataPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRequest_DataPathInference(t *testing.T) {
	query := `{
		users {
			id
			name
		}
	}`

	req := NewRequest(query)
	if req.dataPath != "users" {
		t.Errorf("NewRequest() dataPath = %v, want %v", req.dataPath, "users")
	}

	// Test that manual override works
	req.DataPath("override")
	if req.dataPath != "override" {
		t.Errorf("DataPath() override failed, got = %v, want %v", req.dataPath, "override")
	}
}
