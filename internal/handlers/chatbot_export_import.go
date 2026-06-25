package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const exportVersion = "1.0"

// ─── Payload ─────────────────────────────────────────────────────────────────

type exportPayload struct {
	Version    string               `json:"version"`
	ExportedAt time.Time            `json:"exported_at"`
	Flows      []flowExportItem     `json:"flows"`
	Keywords   []models.KeywordRule `json:"keywords"`
}

type flowExportItem struct {
	models.ChatbotFlow
	Steps []models.ChatbotFlowStep `json:"steps"`
}

// ─── Export ──────────────────────────────────────────────────────────────────

// ExportChatbotData exports flows (with steps) and keyword rules as a JSON file.
//
// POST /api/chatbot/export
// Body (optional JSON): { "flow_ids": ["uuid1"], "keyword_ids": ["uuid2"] }
// Omit or send empty arrays to export everything in the organisation.
func (a *App) ExportChatbotData(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, "chatbot", "export")
	if err != nil {
		return nil
	}

	var req struct {
		FlowIDs    []string `json:"flow_ids"`
		KeywordIDs []string `json:"keyword_ids"`
	}
	_ = r.Decode(&req, "json") // body is optional

	flows, err := a.exportFetchFlows(orgID, req.FlowIDs)
	if err != nil {
		a.Log.Error("export: fetch flows", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch flows", nil, "")
	}

	keywords, err := a.exportFetchKeywords(orgID, req.KeywordIDs)
	if err != nil {
		a.Log.Error("export: fetch keywords", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch keywords", nil, "")
	}

	payload := exportPayload{
		Version:    exportVersion,
		ExportedAt: time.Now().UTC(),
		Flows:      flows,
		Keywords:   keywords,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to serialise export", nil, "")
	}

	r.RequestCtx.SetContentType("application/json")
	r.RequestCtx.Response.Header.Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="whatomate-export-%s.json"`, time.Now().Format("20060102-150405")),
	)
	r.RequestCtx.SetBody(raw)
	return nil
}

func (a *App) exportFetchFlows(orgID uuid.UUID, ids []string) ([]flowExportItem, error) {
	var flows []models.ChatbotFlow
	q := a.DB.Where("organization_id = ? AND deleted_at IS NULL", orgID)
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	}
	if err := q.Find(&flows).Error; err != nil {
		return nil, err
	}

	items := make([]flowExportItem, 0, len(flows))
	for _, f := range flows {
		var steps []models.ChatbotFlowStep
		_ = a.DB.Where("flow_id = ? AND deleted_at IS NULL", f.ID).
			Order("step_order asc").Find(&steps).Error

		// Strip IDs so the export is portable across organisations
		f.ID = uuid.Nil
		f.OrganizationID = uuid.Nil
		f.CreatedByID = nil
		f.UpdatedByID = nil
		for i := range steps {
			steps[i].ID = uuid.Nil
			steps[i].FlowID = uuid.Nil
		}
		items = append(items, flowExportItem{ChatbotFlow: f, Steps: steps})
	}
	return items, nil
}

func (a *App) exportFetchKeywords(orgID uuid.UUID, ids []string) ([]models.KeywordRule, error) {
	var kws []models.KeywordRule
	q := a.DB.Where("organization_id = ? AND deleted_at IS NULL", orgID)
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	}
	if err := q.Find(&kws).Error; err != nil {
		return nil, err
	}
	for i := range kws {
		kws[i].ID = uuid.Nil
		kws[i].OrganizationID = uuid.Nil
		kws[i].CreatedByID = nil
		kws[i].UpdatedByID = nil
	}
	return kws, nil
}

// ─── Import ──────────────────────────────────────────────────────────────────

// ImportChatbotData imports flows and keywords from a JSON produced by ExportChatbotData.
//
// POST /api/chatbot/import
// Body: exported JSON payload
// Query: overwrite=true  → create new record even when a same-name one exists
func (a *App) ImportChatbotData(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, "chatbot", "import")
	if err != nil {
		return nil
	}

	var payload exportPayload
	if err := json.Unmarshal(r.RequestCtx.Request.Body(), &payload); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid import file", nil, "")
	}
	if payload.Version != exportVersion {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			fmt.Sprintf("Unsupported export version: %s", payload.Version), nil, "")
	}

	overwrite := string(r.RequestCtx.QueryArgs().Peek("overwrite")) == "true"

	var (
		importedFlows    int
		importedKeywords int
		skippedFlows    []string
		skippedKeywords []string
	)

	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range payload.Flows {
			name := item.ChatbotFlow.Name

			if !overwrite {
				var n int64
				tx.Model(&models.ChatbotFlow{}).
					Where("organization_id = ? AND name = ? AND deleted_at IS NULL", orgID, name).
					Count(&n)
				if n > 0 {
					skippedFlows = append(skippedFlows, name)
					continue
				}
			}

			newID := uuid.New()
			f := item.ChatbotFlow
			f.ID = newID
			f.OrganizationID = orgID
			f.CreatedByID = &userID
			f.UpdatedByID = &userID

			if err := tx.Create(&f).Error; err != nil {
				return fmt.Errorf("create flow %q: %w", name, err)
			}
			for _, step := range item.Steps {
				step.ID = uuid.New()
				step.FlowID = newID
				if err := tx.Create(&step).Error; err != nil {
					return fmt.Errorf("create step for flow %q: %w", name, err)
				}
			}
			importedFlows++
		}

		for _, kw := range payload.Keywords {
			name := kw.Name
			if !overwrite {
				var n int64
				tx.Model(&models.KeywordRule{}).
					Where("organization_id = ? AND name = ? AND deleted_at IS NULL", orgID, name).
					Count(&n)
				if n > 0 {
					skippedKeywords = append(skippedKeywords, name)
					continue
				}
			}

			kw.ID = uuid.New()
			kw.OrganizationID = orgID
			kw.CreatedByID = &userID
			kw.UpdatedByID = &userID

			if err := tx.Create(&kw).Error; err != nil {
				return fmt.Errorf("create keyword %q: %w", name, err)
			}
			importedKeywords++
		}
		return nil
	})

	if err != nil {
		a.Log.Error("import: transaction failed", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Import failed: "+err.Error(), nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"imported_flows":    importedFlows,
		"imported_keywords": importedKeywords,
		"skipped_flows":     skippedFlows,
		"skipped_keywords":  skippedKeywords,
	})
}
