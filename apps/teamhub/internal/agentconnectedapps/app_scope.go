package agentconnectedapps

import "strings"

// appToolScopes narrow an app to the tools it is allowed to expose. Outlook
// Calendar shares the Outlook toolkit with mail, while Feishu's LARK toolkit
// also contains docs, calendar, email, and other actions; neither connector
// should grant an Agent access to unrelated tools.
var appToolScopes = map[string]func(slug string) bool{
	"outlook_calendar": isOutlookCalendarTool,
	"feishu":           isFeishuMessagingTool,
}

// appAllowsTool reports whether a connection of appKey may search or run slug.
func appAllowsTool(appKey, slug string) bool {
	allows, scoped := appToolScopes[appKey]
	return !scoped || allows(slug)
}

var feishuMessagingTools = map[string]bool{
	"LARK_SEARCH_SEARCH_MESSAGE":              true,
	"LARK_GROUP_CHAT_SEARCH_VISIBLE_CHATS":    true,
	"LARK_GROUP_CHAT_LIST_USER_CHATS":         true,
	"LARK_MESSAGING_SEND_MESSAGE":             true,
}

func isFeishuMessagingTool(slug string) bool {
	return feishuMessagingTools[slug]
}

// outlookCalendarWords are the slug words of Outlook calendar tools; no mail,
// contact or task tool uses them.
var outlookCalendarWords = map[string]bool{
	"CALENDAR": true, "CALENDARS": true, "EVENT": true, "EVENTS": true,
	"SCHEDULE": true, "MEETING": true, "REMINDER": true, "REMINDERS": true,
}

func isOutlookCalendarTool(slug string) bool {
	if !strings.HasPrefix(slug, "OUTLOOK_") {
		return false
	}
	if slug == "OUTLOOK_GET_SUPPORTED_TIME_ZONES" {
		return true
	}
	for _, word := range strings.Split(strings.TrimPrefix(slug, "OUTLOOK_"), "_") {
		if outlookCalendarWords[word] {
			return true
		}
	}
	return false
}
