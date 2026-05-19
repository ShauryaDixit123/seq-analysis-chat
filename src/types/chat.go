package types

const (
	AttachmentKindFASTA            = "fasta"
	AttachmentKindDynamicResult    = "dynamic_result"
	AttachmentKindDotProductResult = "dot_product_result"
)

type ChatAttachment struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	Name        string    `json:"name"`
	Sequence    string    `json:"sequence,omitempty"`
	Size        int64     `json:"size"`
	Kind        string    `json:"kind,omitempty"`
	ProcessType string    `json:"process_type,omitempty"`
	Data        [][][]int `json:"data,omitempty"`
}

type ChatMessage struct {
	ID          string           `json:"id"`
	SessionID   string           `json:"session_id"`
	Text        string           `json:"text"`
	Sender      string           `json:"sender"`
	Timestamp   string           `json:"timestamp"`
	Attachments []ChatAttachment `json:"attachments,omitempty"`
}

type ChatSession struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
