package handler

import (
	"bytes"
	"encoding/json"
	"testing"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildOpenAIImagesMonitorCaptureBody_MultipartProducesUTF8JSONSummary(t *testing.T) {
	body := buildOpenAIImagesMonitorCaptureBody(&service.OpenAIImagesRequest{
		Endpoint:       "/v1/images/edits",
		Multipart:      true,
		Model:          "gpt-image-1",
		Prompt:         "restore this image",
		N:              2,
		HasMask:        true,
		InputImageURLs: []string{"https://example.com/a.png"},
		Uploads: []service.OpenAIImagesUpload{{
			FieldName:   "image",
			FileName:    "input.png",
			ContentType: "image/png",
			Data:        []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x01},
			Width:       1024,
			Height:      768,
		}},
		MaskUpload: &service.OpenAIImagesUpload{
			FieldName:   "mask",
			FileName:    "mask.png",
			ContentType: "image/png",
			Data:        []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x02},
			Width:       1024,
			Height:      768,
		},
	})

	if !utf8.Valid(body) {
		t.Fatalf("capture body is not valid UTF-8: %q", body)
	}
	if !json.Valid(body) {
		t.Fatalf("capture body is not valid JSON: %q", body)
	}
	if bytes.Contains(body, []byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Fatalf("capture body leaked raw binary upload bytes")
	}
	if !bytes.Contains(body, []byte(`"multipart":true`)) {
		t.Fatalf("capture body missing multipart marker: %s", body)
	}
	if !bytes.Contains(body, []byte(`"bytes":6`)) {
		t.Fatalf("capture body missing upload byte counts: %s", body)
	}
}
