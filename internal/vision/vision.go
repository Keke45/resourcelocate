// Package vision turns a photo of unwanted material into a draft listing
// using Claude.
package vision

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/KeKe45/resourcelocate/internal/domain"
)

// Draft is the model's suggestion; the provider confirms or edits it.
type Draft struct {
	Item         string  `json:"item"`
	Material     string  `json:"material"`
	Quantity     string  `json:"quantity"`
	Pieces       int     `json:"pieces"`
	EstKg        float64 `json:"estKg"`
	Condition    string  `json:"condition"`
	Route        string  `json:"route"`
	Hazardous    bool    `json:"hazardous"`
	HazardReason string  `json:"hazardReason"`
	Confidence   string  `json:"confidence"`
}

var ErrRefused = errors.New("the photo could not be read")

type Reader struct {
	client anthropic.Client
	model  string
}

func New(apiKey, model string) *Reader {
	return &Reader{client: anthropic.NewClient(option.WithAPIKey(apiKey)), model: model}
}

var draftSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"item":         map[string]any{"type": "string"},
		"material":     map[string]any{"type": "string", "enum": materialIDs()},
		"quantity":     map[string]any{"type": "string"},
		"pieces":       map[string]any{"type": "integer"},
		"estKg":        map[string]any{"type": "number"},
		"condition":    map[string]any{"type": "string"},
		"route":        map[string]any{"type": "string", "enum": []string{domain.RouteReuse, domain.RouteRepair, domain.RouteRecycle, domain.RouteSpecial}},
		"hazardous":    map[string]any{"type": "boolean"},
		"hazardReason": map[string]any{"type": "string"},
		"confidence":   map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
	},
	"required":             []string{"item", "material", "quantity", "pieces", "estKg", "condition", "route", "hazardous", "hazardReason", "confidence"},
	"additionalProperties": false,
}

func materialIDs() []string {
	ids := make([]string, len(domain.Materials))
	for i, m := range domain.Materials {
		ids[i] = m.ID
	}
	return ids
}

func prompt() string {
	var b strings.Builder
	b.WriteString("The photo shows unwanted material a household, business, workshop or institution in Lusaka, Zambia wants to give away for reuse or recycling. ")
	b.WriteString("Draft a listing for it.\n\nMaterial categories:\n")
	for _, m := range domain.Materials {
		fmt.Fprintf(&b, "- %s: %s (%s)\n", m.ID, m.Label, m.Examples)
	}
	b.WriteString("\nRoutes:\n")
	for _, r := range domain.Routes {
		fmt.Fprintf(&b, "- %s: %s\n", r.ID, r.Description)
	}
	b.WriteString(`
Fields:
- item: a short name a seeker would search for, such as "Hardwood offcuts".
- quantity: what you can count or estimate, such as "2 sacks" or "10 boards".
- pieces: for separate items that people count, such as monitors, desks, chairs or tyres, how many you can see. 0 for loose or bulk material such as offcuts, sawdust or fabric scraps.
- estKg: your best estimate of total weight in kilograms. Say so through confidence when unsure.
- condition: one short phrase on its state.
- route: "reuse" if usable as is, "repair" if it needs fixing or remaking, "recycle" if it needs processing, "special" if only a qualified handler should take it.
- hazardous: true for batteries, chemicals, paint or solvent containers, medical waste, asbestos, fluorescent tubes, or anything leaking or burnt. Then explain in hazardReason and use route "special". Otherwise hazardReason is "".
- confidence: how sure you are about the whole draft.
If the photo does not show a material that could be listed, set item to "", estKg to 0 and confidence to "low".`)
	return b.String()
}

// Read sends the image to Claude and returns a validated draft.
func (r *Reader) Read(ctx context.Context, mediaType string, img []byte) (Draft, error) {
	msg, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(r.model),
		MaxTokens: 4000,
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: draftSchema},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewImageBlockBase64(mediaType, base64.StdEncoding.EncodeToString(img)),
				anthropic.NewTextBlock(prompt()),
			),
		},
	})
	if err != nil {
		return Draft{}, err
	}
	if msg.StopReason == anthropic.StopReasonRefusal {
		return Draft{}, ErrRefused
	}
	var text string
	for _, block := range msg.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			text += tb.Text
		}
	}
	var d Draft
	if err := json.Unmarshal([]byte(text), &d); err != nil {
		return Draft{}, fmt.Errorf("parse draft: %w", err)
	}
	return sanitize(d), nil
}

// sanitize keeps the draft inside the catalog even if the model strays.
func sanitize(d Draft) Draft {
	if _, ok := domain.MaterialByID(d.Material); !ok {
		d.Material = "other"
	}
	if _, ok := domain.RouteByID(d.Route); !ok {
		d.Route = domain.RouteReuse
	}
	if d.Hazardous {
		d.Route = domain.RouteSpecial
	}
	if d.EstKg < 0 {
		d.EstKg = 0
	}
	if d.Pieces < 0 {
		d.Pieces = 0
	}
	return d
}
