package gqlclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_SingleField(t *testing.T) {
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

	client := newClient(server.URL)

	tests := []struct {
		name  string
		query string
		want  interface{}
	}{
		{
			name: "basic query",
			query: `{
				users {
					items {
						id
						name
					}
				}
			}`,
			want: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		},
		{
			name: "named query",
			query: `query GetUsers {
				users {
					items {
						id
						name
					}
				}
			}`,
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
			want: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		},
		{
			name: "anonymous query",
			query: `{
				users {
					items {
						id
						name
					}
				}
			}`,
			want: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewTestRequest(tt.query)

			var resp interface{}
			err := client.RunWithContext(context.Background(), req, &resp)
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

func TestClient_MultipleFields(t *testing.T) {
	mockResponse := map[string]interface{}{
		"data": map[string]interface{}{
			"users": map[string]interface{}{"id": "1", "name": "Alice"},
			"items": []map[string]interface{}{
				{"id": "1", "name": "Alice"},
				{"id": "2", "name": "Bob"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := newClient(server.URL)

	tests := []struct {
		name  string
		query string
		want  interface{}
	}{
		{
			name: "basic query",
			query: `{
				users {
					id
					name
				}
				items {
					id
					name
				}
			}`,
			want: map[string]interface{}{
				"users": map[string]interface{}{
					"id":   "1",
					"name": "Alice",
				},
				"items": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewTestRequest(tt.query)

			var resp interface{}
			err := client.RunWithContext(context.Background(), req, &resp)
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
