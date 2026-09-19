package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// maxChunkChars is the rough size at which release notes get split into
// multiple chunks for a map-reduce summarization pass instead of a single call.
const maxChunkChars = 6000

// telegramFallbackLimit keeps the raw-text fallback under Telegram's message limit.
const telegramFallbackLimit = 3500

type Client struct {
	Host       string
	Model      string
	HTTPClient *http.Client
}

func New(host, model string) *Client {
	return &Client{
		Host:  strings.TrimRight(host, "/"),
		Model: model,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c *Client) generate(promptText string) (string, error) {
	reqBody, err := json.Marshal(generateRequest{
		Model:  c.Model,
		Prompt: promptText,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("llm: encode request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.Host+"/api/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("llm: request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: ollama returned status %d", resp.StatusCode)
	}

	var out generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("llm: decode response: %w", err)
	}

	return strings.TrimSpace(out.Response), nil
}

// Summarize produces a German summary of releaseNotes using systemPrompt and
// context as instructions/background. Long release notes are split by
// heading and summarized in a map step, then combined in a reduce step. If
// the LLM cannot be reached or fails, a plain-text fallback (truncated raw
// release notes plus the release URL) is returned instead of an error, so
// the caller always has something usable to send.
func (c *Client) Summarize(systemPrompt, context, releaseNotes, releaseURL string) string {
	chunks := splitByHeadings(releaseNotes, maxChunkChars)

	if len(chunks) <= 1 {
		summary, err := c.generate(buildPrompt(systemPrompt, context, releaseNotes))
		if err != nil {
			return fallback(releaseNotes, releaseURL)
		}
		return summary
	}

	var partials []string
	for _, chunk := range chunks {
		partial, err := c.generate(buildPrompt(systemPrompt, context, chunk))
		if err != nil {
			return fallback(releaseNotes, releaseURL)
		}
		partials = append(partials, partial)
	}

	reducePrompt := buildReducePrompt(systemPrompt, context, partials)
	final, err := c.generate(reducePrompt)
	if err != nil {
		return fallback(releaseNotes, releaseURL)
	}
	return final
}

func buildPrompt(systemPrompt, context, text string) string {
	var b strings.Builder
	b.WriteString(systemPrompt)
	b.WriteString("\n\n--- Kontext ---\n")
	b.WriteString(context)
	b.WriteString("\n\n--- Release Notes ---\n")
	b.WriteString(text)
	return b.String()
}

func buildReducePrompt(systemPrompt, context string, partials []string) string {
	var b strings.Builder
	b.WriteString(systemPrompt)
	b.WriteString("\n\n--- Kontext ---\n")
	b.WriteString(context)
	b.WriteString("\n\nFasse die folgenden Teil-Zusammenfassungen eines einzelnen Releases zu einer einzigen, zusammenhängenden deutschen Zusammenfassung zusammen. Vermeide Wiederholungen.\n")
	for i, p := range partials {
		fmt.Fprintf(&b, "\n--- Teil %d ---\n%s\n", i+1, p)
	}
	return b.String()
}

func fallback(releaseNotes, releaseURL string) string {
	notes := releaseNotes
	if len(notes) > telegramFallbackLimit {
		notes = notes[:telegramFallbackLimit] + "…"
	}
	return fmt.Sprintf(
		"⚠️ Automatische Zusammenfassung fehlgeschlagen – Original-Release-Notes:\n\n%s\n\nLink: %s",
		notes, releaseURL,
	)
}

// splitByHeadings splits text on markdown headings ("#" ... "######").
// If the whole text is already under maxChars, or no headings are found and
// the text is short, it is returned as a single chunk; otherwise a heading-
// less text is chunked by size.
func splitByHeadings(text string, maxChars int) []string {
	if len(text) <= maxChars {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	var chunks []string
	var current strings.Builder
	hasHeadings := false

	flush := func() {
		if current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			hasHeadings = true
			flush()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	flush()

	if !hasHeadings {
		return chunkBySize(text, maxChars)
	}

	return mergeSmallChunks(chunks, maxChars)
}

// mergeSmallChunks combines consecutive heading-sections so chunks stay
// close to maxChars instead of being one call per heading.
func mergeSmallChunks(chunks []string, maxChars int) []string {
	var merged []string
	var current strings.Builder

	for _, c := range chunks {
		if current.Len() > 0 && current.Len()+len(c) > maxChars {
			merged = append(merged, strings.TrimSpace(current.String()))
			current.Reset()
		}
		current.WriteString(c)
		current.WriteString("\n\n")
	}
	if current.Len() > 0 {
		merged = append(merged, strings.TrimSpace(current.String()))
	}
	return merged
}

func chunkBySize(text string, maxChars int) []string {
	var chunks []string
	for len(text) > maxChars {
		chunks = append(chunks, text[:maxChars])
		text = text[maxChars:]
	}
	if len(text) > 0 {
		chunks = append(chunks, text)
	}
	return chunks
}
