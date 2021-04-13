package chimera

type MessageContent []byte

// Chimera sent message structure
type Message struct {
	ID      string            `json:"id"`
	Content MessageContent    `json:"content"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (m *MessageContent) MarshalJSON() ([]byte, error) {
	return []byte(*m), nil
}

func (m *MessageContent) UnmarshalJSON(data []byte) error {
	*m = data
	return nil
}
