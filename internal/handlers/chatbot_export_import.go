package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// chatbotExportPayload é o formato do arquivo JSON exportado.
type chatbotExportPayload struct {
	Version    string               `json:"version"`
	ExportedAt time.Time            `json:"exported_at"`
	Flows      []models.ChatbotFlow `json:"flows"`
	Keywords   []models.KeywordRule `json:"keywords"`
}

// exportChatbotRequest define quais IDs exportar (vazio = exporta tudo da org).
type exportChatbotRequest struct {
	FlowIDs    []uuid.UUID `json:"flow_ids"`
	KeywordIDs []uuid.UUID `json:"keyword_ids"`
}

// ExportChatbotData exporta flows e keyword rules como JSON para download.
// POST /api/chatbot/export
func (a *App) ExportChatbotData(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceFlowsChatbot, models.ActionRead, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Permission denied", nil, "")
	}

	var req exportChatbotRequest
	_ = r.Decode(&req, "json")

	// Buscar flows da organização
	var flows []models.ChatbotFlow
	flowQ := a.DB.Where("organization_id = ?", orgID)
	if len(req.FlowIDs) > 0 {
		flowQ = flowQ.Where("id IN ?", req.FlowIDs)
	}
	if err := flowQ.Find(&flows).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch flows", nil, "")
	}

	// Buscar keyword rules da organização
	var keywords []models.KeywordRule
	kwQ := a.DB.Where("organization_id = ?", orgID)
	if len(req.KeywordIDs) > 0 {
		kwQ = kwQ.Where("id IN ?", req.KeywordIDs)
	}
	if err := kwQ.Find(&keywords).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch keyword rules", nil, "")
	}

	payload := chatbotExportPayload{
		Version:    "1.0",
		ExportedAt: time.Now().UTC(),
		Flows:      flows,
		Keywords:   keywords,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to serialize export", nil, "")
	}

	// Audit — usa o helper padrão do projeto
	a.logAudit(orgID, userID, "chatbot", orgID, models.AuditActionCreated, nil, nil,
		map[string]any{
			"field":     "export",
			"new_value": fmt.Sprintf("flows:%d keywords:%d", len(flows), len(keywords)),
		},
	)

	r.RequestCtx.Response.Header.Set("Content-Type", "application/json")
	r.RequestCtx.Response.Header.Set("Content-Disposition", `attachment; filename="whatomate-chatbot-export.json"`)
	r.RequestCtx.Response.SetBody(raw)
	return nil
}

// ImportChatbotData importa flows e keyword rules a partir de um JSON exportado.
// POST /api/chatbot/import
func (a *App) ImportChatbotData(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceFlowsChatbot, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Permission denied", nil, "")
	}

	body := r.RequestCtx.Request.Body()
	if len(body) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Request body is required", nil, "")
	}

	var payload chatbotExportPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid import file format", nil, "")
	}

	if !strings.HasPrefix(payload.Version, "1.") {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			fmt.Sprintf("Unsupported export version: %s", payload.Version), nil, "")
	}

	flowsImported := 0
	for _, flow := range payload.Flows {
		flow.ID = uuid.New()
		flow.OrganizationID = orgID
		flow.CreatedByID = &userID
		flow.UpdatedByID = &userID
		flow.CreatedAt = time.Now()
		flow.UpdatedAt = time.Now()
		flow.DeletedAt = gorm.DeletedAt{}

		if err := a.DB.Omit("Organization", "InitialTemplate", "CreatedBy", "UpdatedBy").
			Create(&flow).Error; err != nil {
			a.Log.Error("Failed to import flow", "name", flow.Name, "error", err)
			continue
		}
		flowsImported++
	}

	keywordsImported := 0
	for _, kw := range payload.Keywords {
		kw.ID = uuid.New()
		kw.OrganizationID = orgID
		kw.CreatedByID = &userID
		kw.UpdatedByID = &userID
		kw.CreatedAt = time.Now()
		kw.UpdatedAt = time.Now()
		kw.DeletedAt = gorm.DeletedAt{}

		if err := a.DB.Omit("Organization", "CreatedBy", "UpdatedBy").
			Create(&kw).Error; err != nil {
			a.Log.Error("Failed to import keyword rule", "name", kw.Name, "error", err)
			continue
		}
		keywordsImported++
	}

	// Audit — usa o helper padrão do projeto
	a.logAudit(orgID, userID, "chatbot", orgID, models.AuditActionCreated, nil, nil,
		map[string]any{
			"field":     "import",
			"new_value": fmt.Sprintf("flows:%d keywords:%d", flowsImported, keywordsImported),
		},
	)

	return r.SendEnvelope(map[string]any{
		"flows_imported":    flowsImported,
		"keywords_imported": keywordsImported,
		"message":           fmt.Sprintf("Imported %d flow(s) and %d keyword rule(s)", flowsImported, keywordsImported),
	})
}
