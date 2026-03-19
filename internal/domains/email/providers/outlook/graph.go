package outlook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultGraphBaseURL = "https://graph.microsoft.com/v1.0"

type GraphConfig struct {
	AccessToken string
	BaseURL     string
	Timeout     time.Duration
	Proxy       string
}

type Mailbox struct {
	ID   string
	Name string
}

type MessageSummary struct {
	ID      string
	Subject string
	From    string
	To      string
	Date    time.Time
	Size    int64
	Flags   []string
}

type MessageDetail struct {
	ID       string
	Subject  string
	From     string
	To       string
	Cc       string
	Date     time.Time
	Size     int64
	Flags    []string
	Headers  map[string]string
	TextBody string
	HTMLBody string
}

type graphEnvelope[T any] struct {
	Value []T `json:"value"`
	Count int `json:"@odata.count"`
}

type graphErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type mailFolderEntity struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type emailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type recipient struct {
	EmailAddress emailAddress `json:"emailAddress"`
}

type internetMessageHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type messageBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type messageEntity struct {
	ID                     string                  `json:"id"`
	Subject                string                  `json:"subject"`
	From                   recipient               `json:"from"`
	ToRecipients           []recipient             `json:"toRecipients"`
	CcRecipients           []recipient             `json:"ccRecipients"`
	ReceivedDateTime       string                  `json:"receivedDateTime"`
	Size                   int64                   `json:"size"`
	IsRead                 bool                    `json:"isRead"`
	InternetMessageHeaders []internetMessageHeader `json:"internetMessageHeaders"`
	Body                   messageBody             `json:"body"`
}

func ResolveMailbox(ctx context.Context, cfg GraphConfig, mailbox string) (string, string, error) {
	target := strings.TrimSpace(mailbox)
	if target == "" || strings.EqualFold(target, "inbox") {
		return "inbox", "INBOX", nil
	}

	var direct mailFolderEntity
	query := url.Values{}
	query.Set("$select", "id,displayName")
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, "/me/mailFolders/"+url.PathEscape(target), query, nil, &direct); err == nil {
		if strings.TrimSpace(direct.ID) != "" {
			name := strings.TrimSpace(direct.DisplayName)
			if name == "" {
				name = target
			}
			return strings.TrimSpace(direct.ID), name, nil
		}
	}

	items, err := ListMailboxes(ctx, cfg, 200)
	if err != nil {
		return "", "", err
	}
	for _, item := range items {
		if strings.EqualFold(item.Name, target) || strings.EqualFold(item.ID, target) {
			return item.ID, item.Name, nil
		}
	}
	return "", "", fmt.Errorf("mailbox %q not found", target)
}

func ListMailboxes(ctx context.Context, cfg GraphConfig, top int) ([]Mailbox, error) {
	if top <= 0 {
		top = 200
	}

	query := url.Values{}
	query.Set("$top", fmt.Sprintf("%d", top))
	query.Set("$select", "id,displayName")

	var payload graphEnvelope[mailFolderEntity]
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, "/me/mailFolders", query, nil, &payload); err != nil {
		return nil, err
	}

	items := make([]Mailbox, 0, len(payload.Value))
	for _, raw := range payload.Value {
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(raw.DisplayName)
		if name == "" {
			name = id
		}
		items = append(items, Mailbox{
			ID:   id,
			Name: name,
		})
	}
	return items, nil
}

func ListMessages(ctx context.Context, cfg GraphConfig, mailboxID string, limit int, offset int) ([]MessageSummary, int, error) {
	if strings.TrimSpace(mailboxID) == "" {
		mailboxID = "inbox"
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	query := url.Values{}
	query.Set("$top", fmt.Sprintf("%d", limit))
	query.Set("$skip", fmt.Sprintf("%d", offset))
	query.Set("$count", "true")
	query.Set("$orderby", "receivedDateTime desc")
	query.Set("$select", "id,subject,from,toRecipients,receivedDateTime,size,isRead")

	headers := map[string]string{
		"ConsistencyLevel": "eventual",
	}

	var payload graphEnvelope[messageEntity]
	pathValue := "/me/mailFolders/" + url.PathEscape(mailboxID) + "/messages"
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, pathValue, query, headers, &payload); err != nil {
		return nil, 0, err
	}

	items := make([]MessageSummary, 0, len(payload.Value))
	for _, raw := range payload.Value {
		items = append(items, MessageSummary{
			ID:      strings.TrimSpace(raw.ID),
			Subject: strings.TrimSpace(raw.Subject),
			From:    formatRecipient(raw.From),
			To:      joinRecipients(raw.ToRecipients),
			Date:    parseGraphDate(raw.ReceivedDateTime),
			Size:    raw.Size,
			Flags:   buildFlags(raw.IsRead),
		})
	}

	total := payload.Count
	if total < offset+len(items) {
		total = offset + len(items)
	}
	return items, total, nil
}

func GetMessage(ctx context.Context, cfg GraphConfig, messageID string) (MessageDetail, error) {
	trimmedID := strings.TrimSpace(messageID)
	if trimmedID == "" {
		return MessageDetail{}, errors.New("message id is required")
	}

	query := url.Values{}
	query.Set("$select", "id,subject,from,toRecipients,ccRecipients,receivedDateTime,size,isRead,internetMessageHeaders,body")

	var payload messageEntity
	pathValue := "/me/messages/" + url.PathEscape(trimmedID)
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, pathValue, query, nil, &payload); err != nil {
		return MessageDetail{}, err
	}

	headers := make(map[string]string, len(payload.InternetMessageHeaders))
	for _, item := range payload.InternetMessageHeaders {
		key := strings.TrimSpace(item.Name)
		if key == "" {
			continue
		}
		headers[key] = item.Value
	}

	detail := MessageDetail{
		ID:      strings.TrimSpace(payload.ID),
		Subject: strings.TrimSpace(payload.Subject),
		From:    formatRecipient(payload.From),
		To:      joinRecipients(payload.ToRecipients),
		Cc:      joinRecipients(payload.CcRecipients),
		Date:    parseGraphDate(payload.ReceivedDateTime),
		Size:    payload.Size,
		Flags:   buildFlags(payload.IsRead),
		Headers: headers,
	}

	switch strings.ToLower(strings.TrimSpace(payload.Body.ContentType)) {
	case "text":
		detail.TextBody = payload.Body.Content
	default:
		detail.HTMLBody = payload.Body.Content
	}

	return detail, nil
}

func requestJSON(
	ctx context.Context,
	cfg GraphConfig,
	method string,
	pathValue string,
	query url.Values,
	headers map[string]string,
	output any,
) error {
	accessToken := normalizeAccessToken(cfg.AccessToken)
	if accessToken == "" {
		return errors.New("outlook access_token is required")
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultGraphBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if !strings.HasPrefix(pathValue, "/") {
		pathValue = "/" + pathValue
	}

	targetURL, err := url.Parse(baseURL + pathValue)
	if err != nil {
		return fmt.Errorf("invalid graph base url: %w", err)
	}
	if len(query) > 0 {
		targetURL.RawQuery = query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, method, targetURL.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		request.Header.Set(key, value)
	}

	client, err := buildHTTPClient(cfg.Proxy, cfg.Timeout)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var problem graphErrorResponse
		if json.Unmarshal(body, &problem) == nil {
			code := strings.TrimSpace(problem.Error.Code)
			message := strings.TrimSpace(problem.Error.Message)
			if code != "" || message != "" {
				if code == "" {
					code = fmt.Sprintf("http_%d", response.StatusCode)
				}
				if message == "" {
					message = "graph request failed"
				}
				return fmt.Errorf("%s: %s", code, message)
			}
		}
		return fmt.Errorf("graph request failed with status %d", response.StatusCode)
	}

	if output == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("parse graph response: %w", err)
	}
	return nil
}

func requestJSONWithSelectFallback(
	ctx context.Context,
	cfg GraphConfig,
	method string,
	pathValue string,
	query url.Values,
	headers map[string]string,
	output any,
) error {
	err := requestJSON(ctx, cfg, method, pathValue, query, headers, output)
	if err == nil {
		return nil
	}
	if strings.TrimSpace(query.Get("$select")) == "" {
		return err
	}
	if !strings.Contains(strings.ToLower(err.Error()), "could not find a property named") {
		return err
	}

	retryQuery := cloneValues(query)
	retryQuery.Del("$select")
	return requestJSON(ctx, cfg, method, pathValue, retryQuery, headers, output)
}

func cloneValues(source url.Values) url.Values {
	if len(source) == 0 {
		return url.Values{}
	}
	cloned := make(url.Values, len(source))
	for key, values := range source {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func parseGraphDate(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func formatRecipient(raw recipient) string {
	address := strings.TrimSpace(raw.EmailAddress.Address)
	name := strings.TrimSpace(raw.EmailAddress.Name)
	switch {
	case name != "" && address != "":
		return name + " <" + address + ">"
	case address != "":
		return address
	default:
		return name
	}
}

func joinRecipients(items []recipient) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		recipientValue := formatRecipient(item)
		if recipientValue == "" {
			continue
		}
		values = append(values, recipientValue)
	}
	return strings.Join(values, ", ")
}

func buildFlags(isRead bool) []string {
	if isRead {
		return []string{"\\Seen"}
	}
	return []string{}
}
