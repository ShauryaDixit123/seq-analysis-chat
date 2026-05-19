package handlers

import (
	"atgc/src/methods"
	"atgc/src/methods/fasta"
	"atgc/src/types"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	chatMu             sync.RWMutex
	chatMessages       map[string][]types.ChatMessage
	chatSessions       map[string]types.ChatSession
	chatSessionsMu     sync.RWMutex
	chatSessionFiles   map[string][]types.ChatAttachment
	chatSessionFilesMu sync.RWMutex
)

func init() {
	chatMessages = make(map[string][]types.ChatMessage)
	chatSessions = make(map[string]types.ChatSession)
	chatSessionFiles = make(map[string][]types.ChatAttachment)
	chatSessionFilesMu = sync.RWMutex{}
}

func (h *Handler) ServeChat(c *gin.Context) {
	c.HTML(http.StatusOK, "chat.html", gin.H{
		"title": "Chat",
	})
}

func (h *Handler) ListChatMessages(c *gin.Context) {
	sessionID := c.Query("session_id")
	chatMu.RLock()
	defer chatMu.RUnlock()

	out := make([]types.ChatMessage, len(chatMessages[sessionID]))
	copy(out, chatMessages[sessionID])
	c.JSON(http.StatusOK, gin.H{"messages": out, "session_id": sessionID})
}

func (h *Handler) runDotProduct(sessionID, processType string) [][][]int {
	chatSessionFilesMu.RLock()
	files := append([]types.ChatAttachment(nil), chatSessionFiles[sessionID]...)
	chatSessionFilesMu.RUnlock()
	if len(files) < 2 {
		return nil
	}
	type pair struct {
		seq1, seq2 string
	}
	var pairs []pair
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			pairs = append(pairs, pair{files[i].Sequence, files[j].Sequence})
		}
	}
	results := make([][][]int, len(pairs))
	var wg sync.WaitGroup
	m := methods.NewMatchMethod(
		"Dot Product method",
		time.Now().Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
	)
	for i, p := range pairs {
		wg.Add(1)
		go func(i int, seq1, seq2 string) {
			defer wg.Done()
			results[i] = h.runSequenceAnalysis(m, processType, seq1, seq2)
		}(i, p.seq1, p.seq2)
	}
	wg.Wait()
	return results
}

func (h *Handler) PostChatMessage(c *gin.Context) {
	text := c.PostForm("message")
	sessionID := c.Query("session_id")
	files, err := c.MultipartForm()
	processType := c.DefaultQuery("process_type", "none")
	chatSessionFilesMu.RLock()
	preexistingFiles := chatSessionFiles[sessionID]
	chatSessionFilesMu.RUnlock()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}
	msg := types.ChatMessage{
		ID:        uuid.New().String(),
		Text:      text,
		Sender:    "you",
		Timestamp: time.Now().Format(time.RFC3339),
		SessionID: sessionID,
	}

	resMsg := types.ChatMessage{
		ID:        uuid.New().String(),
		Text:      "YO HOW we doin!",
		Sender:    "assistant",
		Timestamp: time.Now().Format(time.RFC3339),
		SessionID: sessionID,
	}

	if text == "" && len(files.File["files"]) == 0 {
		resMsg.Text = "message or attachment required"
		c.JSON(http.StatusBadRequest, gin.H{
			"messages": []types.ChatMessage{msg, resMsg},
		})
		return
	}
	if err != nil && text == "" {
		resMsg.Text = "failed to parse multipart form"
		c.JSON(http.StatusBadRequest, gin.H{
			"messages": []types.ChatMessage{msg, resMsg},
		})
		return
	}
	var newAttachments []types.ChatAttachment
	if len(files.File["files"]) > 0 {
		if len(files.File["files"]) < 2 && len(preexistingFiles) == 0 {
			fh := files.File["files"][0]
			attachment, err := h.processUploadedFiles(fh, sessionID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process attachment"})
				return
			}
			chatSessionFilesMu.Lock()
			chatSessionFiles[sessionID] = append(chatSessionFiles[sessionID], attachment)
			chatSessionFilesMu.Unlock()
			resMsg.Text = "1 file added to session, please send more to continue.."
			c.JSON(http.StatusAccepted, gin.H{
				"messages": []types.ChatMessage{msg, resMsg},
			})
			return
		}
		for _, fh := range files.File["files"] {
			attachment, err := h.processUploadedFiles(fh, sessionID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process attachment"})
				return
			}
			newAttachments = append(newAttachments, attachment)
		}
		chatSessionFilesMu.Lock()
		chatSessionFiles[sessionID] = append(chatSessionFiles[sessionID], newAttachments...)
		chatSessionFilesMu.Unlock()
	}
	if processType == "dot_product" && processType != "none" {
		dotproductResults := h.runDotProduct(sessionID, processType)
		if len(dotproductResults) == 0 {
			resMsg.Text = "Upload at least two FASTA files to run sequence analysis."
		} else {
			resMsg.Text = fmt.Sprintf(
				"%s complete — %d comparison(s). Scatter plot shows DP table scores (row × column).",
				processTypeLabel(processType),
				len(dotproductResults),
			)
			resMsg.Attachments = append(resMsg.Attachments, types.ChatAttachment{
				ID:          uuid.New().String(),
				SessionID:   sessionID,
				Name:        processTypeLabel(processType) + " result",
				Kind:        types.AttachmentKindDotProductResult,
				ProcessType: processType,
				Data:        dotproductResults,
				Size:        int64(len(dotproductResults)),
			})
		}
	}
	if processType == "dynamic_programming" && processType != "none" {
		dynamicProgrammingResults := h.runDynamicProgrammingMatch(sessionID, processType)
		if len(dynamicProgrammingResults) == 0 {
			resMsg.Text = "Upload at least two FASTA files to run sequence analysis."
		} else {
			resMsg.Text = fmt.Sprintf(
				"%s complete — %d comparison(s). Scatter plot shows DP table scores (row × column).",
				processTypeLabel(processType),
				len(dynamicProgrammingResults),
			)
			resMsg.Attachments = append(resMsg.Attachments, types.ChatAttachment{
				ID:          uuid.New().String(),
				SessionID:   sessionID,
				Name:        processTypeLabel(processType) + " result",
				Kind:        types.AttachmentKindDynamicResult,
				ProcessType: processType,
				Data:        dynamicProgrammingResults,
				Size:        int64(len(dynamicProgrammingResults)),
			})
		}
	}
	chatMu.Lock()
	chatMessages[sessionID] = append(chatMessages[sessionID], msg)
	chatMessages[sessionID] = append(chatMessages[sessionID], resMsg)
	chatMu.Unlock()

	c.JSON(http.StatusCreated, gin.H{"messages": []types.ChatMessage{msg, resMsg}})
}

func processTypeLabel(processType string) string {
	switch processType {
	case "global_alignment":
		return "Global alignment"
	case "local_alignment":
		return "Local alignment"
	case "dynamic_programming":
		return "Dynamic programming"
	default:
		return "Sequence analysis"
	}
}

func (h *Handler) runDynamicProgrammingMatch(sessionID, processType string) [][][]int {
	chatSessionFilesMu.RLock()
	files := append([]types.ChatAttachment(nil), chatSessionFiles[sessionID]...)
	chatSessionFilesMu.RUnlock()

	if len(files) < 2 {
		return nil
	}

	type pair struct {
		seq1, seq2 string
	}
	var pairs []pair
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			pairs = append(pairs, pair{files[i].Sequence, files[j].Sequence})
		}
	}

	results := make([][][]int, len(pairs))
	var wg sync.WaitGroup
	m := methods.NewMatchMethod(
		"Dynamic Programming method",
		time.Now().Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
	)

	for i, p := range pairs {
		wg.Add(1)
		go func(i int, seq1, seq2 string) {
			defer wg.Done()
			results[i] = h.runSequenceAnalysis(m, processType, seq1, seq2)
		}(i, p.seq1, p.seq2)
	}
	wg.Wait()
	return results
}

func (h *Handler) runSequenceAnalysis(
	m *methods.Method,
	processType, seq1, seq2 string,
) [][]int {
	body := types.MethodRequestBody{
		Sequence1: seq1,
		Sequence2: seq2,
	}
	switch processType {
	case "global_alignment", "local_alignment", "dynamic_programming":
		return m.DyanmicProgrammingMatch(h.Ctx, body)
	case "dot_product":
		return m.DotProduct(h.Ctx, body)
	default:
		return m.DyanmicProgrammingMatch(h.Ctx, body)
	}
}

func (h *Handler) InitChatSession(c *gin.Context) {
	sessionID := uuid.New().String()
	session := types.ChatSession{
		ID:        sessionID,
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	chatMu.Lock()
	chatSessions[sessionID] = session
	chatMu.Unlock()
	c.JSON(http.StatusCreated, gin.H{"session": session})
}

func (h *Handler) processUploadedFiles(
	file *multipart.FileHeader,
	sessionID string,
) (types.ChatAttachment, error) {
	if !fasta.IsFASTA(file.Filename) {
		return types.ChatAttachment{}, fmt.Errorf("unsupported file type %q (expected .fa / .fasta)", filepath.Ext(file.Filename))
	}

	src, err := file.Open()
	if err != nil {
		return types.ChatAttachment{}, fmt.Errorf("open upload: %w", err)
	}
	defer src.Close()

	record, err := fasta.First(src)
	if err != nil {
		return types.ChatAttachment{}, fmt.Errorf("parse FASTA: %w", err)
	}

	return types.ChatAttachment{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Name:      file.Filename,
		Kind:      types.AttachmentKindFASTA,
		Sequence:  record.Sequence,
		Size:      file.Size,
	}, nil
}

func (h *Handler) GetChatMessages(c *gin.Context) {
	sessionID := c.Param("session_id")
	chatMu.RLock()
	defer chatMu.RUnlock()
	messages := chatMessages[sessionID]
	if messages == nil {
		messages = []types.ChatMessage{}
	}
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}
