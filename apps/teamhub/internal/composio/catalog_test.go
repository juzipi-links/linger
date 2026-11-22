package composio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCanStartBeforeAnyManagedAuthConfigExists(t *testing.T) {
	if _, err := New(Config{APIKey: "test-key", EnvironmentKey: "test"}); err != nil {
		t.Fatalf("a project key must support catalog discovery and first-use auth provisioning: %v", err)
	}
}

func TestCustomOAuthAuthConfigUsesCredentialsAndCredentialFingerprint(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v3.1/auth_configs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"toolkit": map[string]any{"slug": "lark"},
			"auth_config": map[string]any{
				"id": "ac_lark", "name": "feishu-auth", "auth_scheme": "OAUTH2",
				"is_composio_managed": false, "status": "ENABLED",
				"toolkit": map[string]any{"slug": "lark"},
			},
		})
	}))
	defer server.Close()

	credentials := map[string]map[string]string{
		"lark": {
			"client_id": "cli_feishu",
			"client_secret": "secret-value",
			"oauth_redirect_uri": LarkOAuthRedirectURI,
		},
	}
	client, err := New(Config{
		BaseURL: server.URL, APIKey: "test-key", EnvironmentKey: "test",
		HTTPClient: server.Client(), CustomAuthCredentials: credentials,
	})
	if err != nil {
		t.Fatal(err)
	}
	name := client.AuthConfigName("lark", "OAUTH2")
	if name == "" || name == "linger:test:lark:OAUTH2:v1" {
		t.Fatalf("custom credentials need a versioned auth config name: %q", name)
	}
	if _, err := client.CreateAuthConfig(t.Context(), "lark", "feishu-auth", "OAUTH2", false); err != nil {
		t.Fatal(err)
	}
	authConfig := received["auth_config"].(map[string]any)
	gotCredentials := authConfig["credentials"].(map[string]any)
	if authConfig["type"] != "use_custom_auth" || authConfig["authScheme"] != "OAUTH2" ||
		gotCredentials["client_id"] != "cli_feishu" ||
		gotCredentials["client_secret"] != "secret-value" ||
		gotCredentials["oauth_redirect_uri"] != LarkOAuthRedirectURI {
		t.Fatal("custom OAuth credentials were not sent in the Composio auth config")
	}
	if !client.HasCustomOAuthCredentials("lark") {
		t.Fatal("configured LARK OAuth credentials were not recognized")
	}

	rotated, err := New(Config{
		APIKey: "test-key", EnvironmentKey: "test",
		CustomAuthCredentials: map[string]map[string]string{
			"lark": {"client_id": "cli_feishu", "client_secret": "rotated-secret", "oauth_redirect_uri": LarkOAuthRedirectURI},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rotated.AuthConfigName("lark", "OAUTH2") == name {
		t.Fatal("rotating the client secret must select a new auth config")
	}
	if client.WithAPIKey("replacement").AuthConfigName("lark", "OAUTH2") != name {
		t.Fatal("changing the Composio API key must preserve the custom OAuth credentials")
	}
}

func TestFindAuthConfigRequiresCompletePagesAndSelectsLowestMatchingID(t *testing.T) {
	for _, incomplete := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "incomplete"}[incomplete], func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.URL.Query().Get("cursor") == "last" && incomplete {
					_, _ = w.Write([]byte(`{"unexpected":"missing items"}`))
					return
				}
				item := AuthConfig{ID: "ac_z", Name: "stable-name", AuthScheme: "OAUTH2", Managed: true, Status: "ENABLED"}
				item.Toolkit.Slug = "hubspot"
				items, next := []AuthConfig{item}, "last"
				if r.URL.Query().Get("cursor") == "last" {
					item.ID = "ac_b"
					items, next = []AuthConfig{item}, ""
					item.ID, item.Name = "ac_a", "foreign-name"
					items = append(items, item)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "next_cursor": next})
			}))
			t.Cleanup(server.Close)
			client, err := newTestClient(Config{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()})
			if err != nil {
				t.Fatal(err)
			}
			config, found, err := client.FindAuthConfig(t.Context(), "hubspot", "stable-name", "OAUTH2", true)
			if requests != 2 {
				t.Fatalf("must read all pages, got %d requests", requests)
			}
			if incomplete {
				if err == nil || found || config.ID != "" {
					t.Fatalf("partial matches cannot select an auth config: %+v %t %v", config, found, err)
				}
			} else if err != nil || !found || config.ID != "ac_b" {
				t.Fatalf("expected lowest matching ID across pages: %+v %t %v", config, found, err)
			}
		})
	}
}
