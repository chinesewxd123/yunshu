package service

import "testing"

func TestMergeAssigneeEmailsStrictPriority(t *testing.T) {
	t.Parallel()
	payload := map[string]interface{}{
		"assignee_emails": []string{"rule@x.com"},
	}
	out := mergeAssigneeEmails([]string{"channel@x.com"}, payload)
	if len(out) != 1 || out[0] != "rule@x.com" {
		t.Fatalf("assignee only: %v", out)
	}
}

func TestMergeAssigneeEmailsChannelFallback(t *testing.T) {
	t.Parallel()
	out := mergeAssigneeEmails([]string{"A@x.com", "a@x.com"}, map[string]interface{}{})
	if len(out) != 1 || out[0] != "a@x.com" {
		t.Fatalf("got %v", out)
	}
}

func TestPayloadHasAssigneeEmails(t *testing.T) {
	t.Parallel()
	if payloadHasAssigneeEmails(nil) {
		t.Fatal("nil payload")
	}
	if !payloadHasAssigneeEmails(map[string]interface{}{
		"assignee_emails": []string{"a@x.com"},
	}) {
		t.Fatal("expected true")
	}
}

func TestMergeAssigneeEmailsWithReceiverGroup(t *testing.T) {
	t.Parallel()
	payload := map[string]interface{}{
		"receiver_group_emails": []string{"cc@x.com"},
	}
	out := mergeAssigneeEmailsWithReceiverGroup([]string{"to@x.com"}, payload)
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
	// 有 assignee 时不合并接收组抄送
	payload2 := map[string]interface{}{
		"assignee_emails":       []string{"rule@x.com"},
		"receiver_group_emails": []string{"cc@x.com"},
	}
	out2 := mergeAssigneeEmailsWithReceiverGroup([]string{"to@x.com"}, payload2)
	// 有 assignee 时本函数不合并 receiver_group_emails，保持原通道收件人
	if len(out2) != 1 || out2[0] != "to@x.com" {
		t.Fatalf("assignee present skips group merge: %v", out2)
	}
}
