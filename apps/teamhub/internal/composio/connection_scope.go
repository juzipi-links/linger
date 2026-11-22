package composio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"sort"
	"strings"
)

var ErrConnectionScope = errors.New("connected app does not belong to this environment")
var environmentKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var appKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,119}$`)

func ValidAppKey(value string) bool { return appKeyPattern.MatchString(value) }

type ConnectionBinding struct {
	UserID, ToolkitSlug, AuthConfigID string
}

// ConnectionAuthority resolves the immutable upstream receipt from TeamHub.
// It must reject receipts from another deployment before any provider request.
type ConnectionAuthority func(context.Context, string) (ConnectionBinding, error)

func (c *Client) SetConnectionAuthority(authority ConnectionAuthority) {
	c.connectionAuthority = authority
}

func (c *Client) UserIDForAccount(accountID string) string {
	return "linger:" + c.environmentKey + ":account:" + accountID
}

// AuthConfigName is the stable name TeamHub finds or creates the auth config
// of toolkit under.
func (c *Client) AuthConfigName(toolkit, scheme string) string {
	version := "v1"
	if credentials := c.customAuthCredentials[toolkit]; len(credentials) > 0 {
		keys := make([]string, 0, len(credentials))
		for key := range credentials {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var fingerprint strings.Builder
		for _, key := range keys {
			fingerprint.WriteString(key)
			fingerprint.WriteByte(0)
			fingerprint.WriteString(credentials[key])
			fingerprint.WriteByte(0)
		}
		digest := sha256.Sum256([]byte(fingerprint.String()))
		version += "-" + hex.EncodeToString(digest[:8])
	}
	return "linger:" + c.environmentKey + ":" + toolkit + ":" + scheme + ":" + version
}

func (c *Client) authorizeConnection(ctx context.Context, upstreamID string) (ConnectionBinding, error) {
	if c.connectionAuthority == nil {
		return ConnectionBinding{}, ErrConnectionScope
	}
	return c.connectionAuthority(ctx, upstreamID)
}
