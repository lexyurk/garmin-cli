package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
)

// PostMultipartFile posts one in-memory file and decodes a JSON response.
func (c *Client) PostMultipartFile(ctx context.Context, path, field, filename, fileContentType string, data []byte, out any) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	h := make(textproto.MIMEHeader)
	field = sanitizeMultipartName(field)
	filename = sanitizeMultipartName(filename)
	h["Content-Disposition"] = []string{`form-data; name="` + field + `"; filename="` + filename + `"`}
	if fileContentType != "" {
		h["Content-Type"] = []string{fileContentType}
	}
	part, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	resp, err := c.Do(ctx, http.MethodPost, path, nil, &body, mw.FormDataContentType())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return err
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func sanitizeMultipartName(s string) string {
	return strings.NewReplacer("\r", "_", "\n", "_", `"`, "_").Replace(s)
}
