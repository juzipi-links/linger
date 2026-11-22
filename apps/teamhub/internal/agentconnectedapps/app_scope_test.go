package agentconnectedapps

import "testing"

func TestFeishuAllowsOnlyMessagingTools(t *testing.T) {
	for slug, want := range map[string]bool{
		"LARK_SEARCH_SEARCH_MESSAGE":           true,
		"LARK_GROUP_CHAT_SEARCH_VISIBLE_CHATS": true,
		"LARK_GROUP_CHAT_LIST_USER_CHATS":      true,
		"LARK_MESSAGING_SEND_MESSAGE":          true,
		"LARK_DOCS_SEARCH_FOR_A_DOCUMENT":      false,
		"LARK_CALENDAR_SEARCH_FOR_EVENTS":      false,
		"LARK_EMAIL_SEND_MESSAGE":              false,
		"SLACK_SEND_MESSAGE":                   false,
	} {
		if got := appAllowsTool("feishu", slug); got != want {
			t.Errorf("appAllowsTool(feishu, %s) = %t, want %t", slug, got, want)
		}
	}
	if !appAllowsTool("slack", "SLACK_SEND_MESSAGE") {
		t.Fatal("the Feishu scope must not change Slack's existing toolkit")
	}
}
