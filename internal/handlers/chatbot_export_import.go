package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// ---------------------------------------------------------------------------
// Export/Import payload types
// ---------------------------------------------------------------------------

// ExportBundle is the top-level envelope written to / read from the JSON file.
type ExportBundle struct {
	Version    string            `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	Flows      []ChatbotFlowDump `json:"flows"`
	Keywords   []KeywordRuleDump `json:"keywords"`
}

// ChatbotFlowDump carries all exportable fields of a ChatbotFlow (no IDs).
type ChatbotFlowDump struct {
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	TriggerKeywords   []string       `json:"trigger_keywords"`
	InitialMessage    string         `json:"initial_message"`
	CompletionMessage string         `json:"completion_message"`
	OnCompleteAction  string         `json:"on_complete_action"`
	CompletionConfig  map[string]any `json:"completion_config"`
	PanelConfig       map[string]any `json:"panel_config"`
	Graph             map[string]any `json:"graph"`
	Enabled           bool           `json:"enabled"`
}

// KeywordRuleDump carries all exportable fields of a KeywordRule (no IDs).
type KeywordRuleDump struct {
	Name            string              `json:"name"`
	Keywords        []string            `json:"keywords"`
	MatchType       models.MatchType    `json:"match_type"`
	ResponseType    models.ResponseType `json:"response_type"`
	ResponseContent map[string]any      `json:"response_content"`
	Priority        int                 `json:"priority"`
	Enabled         bool                `json:"enabled"`
}

// ---------------------------------------------------------------------------
// ExportChatbotData  POST /api/chatbot/export
//
// Optional body: { "flow_ids": ["uuid",...], "keyword_ids": ["uuid",...] }
// Empty/omitted arrays → export ALL records for the org.
// Response: JSON attachment (Content-Disposition: attachment).
// ---------------------------------------------------------------------------
func (a *App) ExportChatbotData(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !a.HasPermission(userID, models.ResourceFlowsChatbot, models.ActionRead, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Permission denied", nil, "")
	}

	var req struct {
		FlowIDs    []string `json:"flow_ids"`
		KeywordIDs []string `json:"keyword_ids"`
	}
	// Body is optional — ignore decode errors (empty body = export all)
	_ = json.Unmarshal(r.RequestCtx.PostBody(), &req)

	bundle := ExportBundle{
		Version:    "1.0",
		ExportedAt: time.Now().UTC(),
	}

	// ---- chatbot flows ----
	flowQ := a.DB.Model(&models.ChatbotFlow{}).Where("organization_id = ?", orgID)
	if len(req.FlowIDs) > 0 {
		flowQ = flowQ.Where("id IN ?", req.FlowIDs)
	}
	var flows []models.ChatbotFlow
	if err := flowQ.Find(&flows).Error; err != nil {
		a.Log.Error("Export: failed to fetch chatbot flows", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch flows", nil, "")
	}
	for _, f := range flows {
		var cc, pc, g map[string]any
		_ = json.Unmarshal(exportMarshal(f.CompletionConfig), &cc)
		_ = json.Unmarshal(exportMarshal(f.PanelConfig), &pc)
		_ = json.Unmarshal(exportMarshal(f.Graph), &g)
		bundle.Flows = append(bundle.Flows, ChatbotFlowDump{
			Name:              f.Name,
			Description:       f.Description,
			TriggerKeywords:   f.TriggerKeywords,
			InitialMessage:    f.InitialMessage,
			CompletionMessage: f.CompletionMessage,
			OnCompleteAction:  f.OnCompleteAction,
			CompletionConfig:  cc,
			PanelConfig:       pc,
			Graph:             g,
			Enabled:           f.IsEnabled,
		})
	}

	// ---- keyword rules ----
	kwQ := a.DB.Model(&models.KeywordRule{}).Where("organization_id = ?", orgID)
	if len(req.KeywordIDs) > 0 {
		kwQ = kwQ.Where("id IN ?", req.KeywordIDs)
	}
	var rules []models.KeywordRule
	if err := kwQ.Find(&rules).Error; err != nil {
		a.Log.Error("Export: failed to fetch keyword rules", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch keyword rules", nil, "")
	}
	for _, kw := range rules {
		var rc map[string]any
		_ = json.Unmarshal(exportMarshal(kw.ResponseContent), &rc)
		bundle.Keywords = append(bundle.Keywords, KeywordRuleDump{
			Name:            kw.Name,
			Keywords:        kw.Keywords,
			MatchType:       kw.MatchType,
			ResponseType:    kw.ResponseType,
			ResponseContent: rc,
			Priority:        kw.Priority,
			Enabled:         kw.IsEnabled,
		})
	}

	payload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to serialize export", nil, "")
	}

	r.RequestCtx.Response.Header.Set("Content-Type", "application/json")
	r.RequestCtx.Response.Header.Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="whatomate-export-%s.json"`, time.Now().Format("2006-01-02")),
	)
	r.RequestCtx.SetBody(payload)
	return nil
}

// ---------------------------------------------------------------------------
// ImportChatbotData  POST /api/chatbot/import
//
// Body: JSON file produced by ExportChatbotData.
// All records receive fresh UUIDs — original IDs are never reused.
// ---------------------------------------------------------------------------
func (a *App) ImportChatbotData(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !a.HasPermission(userID, models.ResourceFlowsChatbot, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Permission denied", nil, "")
	}

	var bundle ExportBundle
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &bundle); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid import file", nil, "")
	}
	if bundle.Version != "1.0" {
		return r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			fmt.Sprintf("Unsupported export version: %s", bundle.Version), nil, "")
	}

	var (
		importedFlows    int
		importedKeywords int
		importErrors     []string
	)

	// ---- import chatbot flows ----
	for _, item := range bundle.Flows {
		if item.Name == "" {
			importErrors = append(importErrors, "skipped flow with empty name")
			continue
		}
		flow := models.ChatbotFlow{
			BaseModel:         models.BaseModel{ID: uuid.New()},
			OrganizationID:    orgID,
			Name:              item.Name,
			Description:       item.Description,
			TriggerKeywords:   item.TriggerKeywords,
			InitialMessage:    item.InitialMessage,
			CompletionMessage: item.CompletionMessage,
			OnCompleteAction:  item.OnCompleteAction,
			CompletionConfig:  models.JSONB(item.CompletionConfig),
			PanelConfig:       models.JSONB(item.PanelConfig),
			Graph:             models.JSONB(item.Graph),
			IsEnabled:         item.Enabled,
			CreatedByID:       &userID,
			UpdatedByID:       &userID,
		}
		if err := a.DB.Create(&flow).Error; err != nil {
			a.Log.Error("Import: failed to create flow", "name", item.Name, "error", err)
			importErrors = append(importErrors, fmt.Sprintf("flow %q: %v", item.Name, err))
			continue
		}
		a.logAudit(orgID, userID, "chatbot_flow", flow.ID, models.AuditActionCreated, nil, &flow)
		importedFlows++
	}

	// ---- import keyword rules ----
	for _, item := range bundle.Keywords {
		if len(item.Keywords) == 0 {
			importErrors = append(importErrors, fmt.Sprintf("skipped keyword rule %q: no keywords", item.Name))
			continue
		}
		if item.MatchType == "" {
			item.MatchType = models.MatchTypeContains
		}
		if item.ResponseType == "" {
			item.ResponseType = models.ResponseTypeText
		}
		if item.Name == "" {
			item.Name = item.Keywords[0]
		}
		rule := models.KeywordRule{
			BaseModel:       models.BaseModel{ID: uuid.New()},
			OrganizationID:  orgID,
			Name:            item.Name,
			Keywords:        item.Keywords,
			MatchType:       item.MatchType,
			ResponseType:    item.ResponseType,
			ResponseContent: models.JSONB(item.ResponseContent),
			Priority:        item.Priority,
			IsEnabled:       item.Enabled,
			CreatedByID:     &userID,
			UpdatedByID:     &userID,
		}
		if err := a.DB.Create(&rule).Error; err != nil {
			a.Log.Error("Import: failed to create keyword rule", "name", item.Name, "error", err)
			importErrors = append(importErrors, fmt.Sprintf("keyword %q: %v", item.Name, err))
			continue
		}
		a.logAudit(orgID, userID, "keyword_rule", rule.ID, models.AuditActionCreated, nil, &rule)
		importedKeywords++
	}

	a.InvalidateChatbotFlowsCache(orgID)
	a.InvalidateKeywordRulesCache(orgID)

	return r.SendEnvelope(map[string]any{
		"imported_flows":    importedFlows,
		"imported_keywords": importedKeywords,
		"errors":            importErrors,
		"message": fmt.Sprintf(
			"Import complete: %d flow(s), %d keyword rule(s)",
			importedFlows, importedKeywords,
		),
	})
}

// exportMarshal encodes v as JSON bytes; returns nil on error.
func exportMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
