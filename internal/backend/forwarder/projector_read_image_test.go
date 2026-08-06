package forwarder

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"cursor/gen/agentv1"
)

func TestProjectPromptReplayAttachesReadImageToToolMessage(t *testing.T) {
	testCases := []struct {
		name        string
		imageData   []byte
		fileSize    uint32
		wantSummary string
	}{
		{
			name:        "small image omits result json base64",
			imageData:   append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...),
			fileSize:    391998,
			wantSummary: `read image path="diagram.png" mime=image/png bytes=72 file_size=391998`,
		},
		{
			name:        "large image omits replay truncation notice",
			imageData:   append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, projectedReadReplayLimit)...),
			fileSize:    uint32(projectedReadReplayLimit + 8),
			wantSummary: fmt.Sprintf(`read image path="diagram.png" mime=image/png bytes=%d`, projectedReadReplayLimit+8),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			toolCall := &agentv1.ToolCall{
				Tool: &agentv1.ToolCall_ReadToolCall{
					ReadToolCall: &agentv1.ReadToolCall{
						Args: &agentv1.ReadToolArgs{Path: "diagram.png"},
						Result: &agentv1.ReadToolResult{
							Result: &agentv1.ReadToolResult_Success{
								Success: &agentv1.ReadToolSuccess{
									FileSize: testCase.fileSize,
									Path:     "diagram.png",
									Output:   &agentv1.ReadToolSuccess_Data{Data: testCase.imageData},
								},
							},
						},
					},
				},
			}
			encodedToolCall, err := protojson.Marshal(toolCall)
			if err != nil {
				t.Fatalf("marshal read tool call: %v", err)
			}
			conversation := &ConversationFile{
				ConversationID: "conversation-1",
				NextTurnSeq:    2,
				Entries: []HistoryEntry{
					newToolResultEntry(1, "request-1", "call-1", "Read", `{"path":"diagram.png"}`, fmt.Sprintf("read binary bytes=%d", len(testCase.imageData)), "", encodedToolCall),
				},
			}

			messages, err := NewHistoryProjector().ProjectPromptReplay(conversation)
			if err != nil {
				t.Fatalf("ProjectPromptReplay() error = %v", err)
			}
			if len(messages) != 2 {
				t.Fatalf("message count = %d, want assistant tool call and tool result", len(messages))
			}
			toolMessage := messages[1]
			if toolMessage.Role != "tool" || toolMessage.ToolCallID != "call-1" || toolMessage.Name != "Read" {
				t.Fatalf("tool message metadata = %#v", toolMessage)
			}
			if toolMessage.Content != testCase.wantSummary {
				t.Fatalf("tool content = %q, want %q", toolMessage.Content, testCase.wantSummary)
			}
			for _, forbidden := range []string{`"data"`, "iVBOR", "base64", "tool result replay truncated"} {
				if strings.Contains(toolMessage.Content, forbidden) {
					t.Fatalf("tool content contains %q: %q", forbidden, toolMessage.Content)
				}
			}
			if len(toolMessage.ContentParts) != 2 {
				t.Fatalf("tool content parts = %d, want text and image", len(toolMessage.ContentParts))
			}
			if toolMessage.ContentParts[0].Type != "text" || toolMessage.ContentParts[0].Text != testCase.wantSummary {
				t.Fatalf("tool text part = %#v", toolMessage.ContentParts[0])
			}
			image := toolMessage.ContentParts[1].Image
			if toolMessage.ContentParts[1].Type != "image" || image == nil {
				t.Fatalf("tool image part = %#v", toolMessage.ContentParts[1])
			}
			if image.MIMEType != "image/png" || image.Path != "diagram.png" || !bytes.Equal(image.Data, testCase.imageData) {
				t.Fatalf("tool image = %#v", image)
			}
		})
	}
}
