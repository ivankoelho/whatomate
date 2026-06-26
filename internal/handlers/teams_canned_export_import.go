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

// ─────────────────────────────────────────────────────────
// Payload container
// ─────────────────────────────────────────────────────────

// teamsAndCannedExportPayload é o envelope JSON exportado.
type teamsAndCannedExportPayload struct {
	Version         string                  `json:"version"`
	ExportedAt      time.Time               `json:"exported_at"`
	Teams           []models.Team           `json:"teams"`
	CannedResponses []models.CannedResponse `json:"canned_responses"`
}

// ─────────────────────────────────────────────────────────
// Export
// ─────────────────────────────────────────────────────────

// ExportTeamsAndCanned exporta equipes e respostas rápidas como JSON.
// POST /api/teams-canned/export
func (a *App) ExportTeamsAndCanned(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceTeams, "read")
	if err != nil {
		return err
	}

	var teams []models.Team
	if err := a.DB.
		Where("organization_id = ?", orgID).
		Preload("Members").
		Find(&teams).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch teams", nil, "")
	}

	var canned []models.CannedResponse
	if err := a.DB.
		Where("organization_id = ?", orgID).
		Find(&canned).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch canned responses", nil, "")
	}

	payload := teamsAndCannedExportPayload{
		Version:         "1.0",
		ExportedAt:      time.Now().UTC(),
		Teams:           teams,
		CannedResponses: canned,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to serialize export", nil, "")
	}

	a.logAudit(orgID, userID, models.ResourceTeams, orgID, models.AuditActionCreated, nil, nil,
		map[string]any{
			"field":     "export",
			"new_value": fmt.Sprintf("teams:%d canned_responses:%d", len(teams), len(canned)),
		},
	)

	r.RequestCtx.Response.Header.Set("Content-Type", "application/json")
	r.RequestCtx.Response.Header.Set(
		"Content-Disposition",
		`attachment; filename="whatomate-teams-canned-export.json"`,
	)
	r.RequestCtx.Response.SetBody(raw)
	return nil
}

// ─────────────────────────────────────────────────────────
// Import
// ─────────────────────────────────────────────────────────

// ImportTeamsAndCanned importa equipes e respostas rápidas a partir de JSON exportado.
// POST /api/teams-canned/import
func (a *App) ImportTeamsAndCanned(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceTeams, "write")
	if err != nil {
		return err
	}

	body := r.RequestCtx.Request.Body()
	if len(body) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Request body is required", nil, "")
	}

	var payload teamsAndCannedExportPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid import file format", nil, "")
	}

	teamsImported := 0
	for _, team := range payload.Teams {
		newTeam := models.Team{
			BaseModel: models.BaseModel{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				DeletedAt: gorm.DeletedAt{},
			},
			OrganizationID:      orgID,
			Name:                team.Name,
			Description:         team.Description,
			AssignmentStrategy:  team.AssignmentStrategy,
			PerAgentTimeoutSecs: team.PerAgentTimeoutSecs,
			IsActive:            team.IsActive,
			CreatedByID:         &userID,
			UpdatedByID:         &userID,
		}

		if err := a.DB.
			Omit("Organization", "Members", "CreatedBy", "UpdatedBy").
			Create(&newTeam).Error; err != nil {
			a.Log.Error("Failed to import team", "name", team.Name, "error", err)
			continue
		}

		// Importar membros (só se o usuário existir nessa org)
		for _, member := range team.Members {
			var uo models.UserOrganization
			if a.DB.Where("user_id = ? AND organization_id = ?", member.UserID, orgID).
				First(&uo).Error != nil {
				continue
			}
			newMember := models.TeamMember{
				BaseModel: models.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				TeamID: newTeam.ID,
				UserID: member.UserID,
				Role:   member.Role,
			}
			_ = a.DB.Omit("Team", "User").Create(&newMember).Error
		}

		teamsImported++
	}

	cannedImported := 0
	for _, cr := range payload.CannedResponses {
		newCR := models.CannedResponse{
			BaseModel: models.BaseModel{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				DeletedAt: gorm.DeletedAt{},
			},
			OrganizationID: orgID,
			Name:           cr.Name,
			Shortcut:       cr.Shortcut,
			Content:        cr.Content,
			Category:       cr.Category,
			IsActive:       cr.IsActive,
			Buttons:        cr.Buttons,
			CreatedByID:    userID,
			UsageCount:     0,
		}

		if err := a.DB.Omit("Organization", "CreatedBy").Create(&newCR).Error; err != nil {
			a.Log.Error("Failed to import canned response", "name", cr.Name, "error", err)
			continue
		}
		cannedImported++
	}

	a.logAudit(orgID, userID, models.ResourceTeams, orgID, models.AuditActionCreated, nil, nil,
		map[string]any{
			"field":     "import",
			"new_value": fmt.Sprintf("teams:%d canned_responses:%d", teamsImported, cannedImported),
		},
	)

	return r.SendEnvelope(map[string]any{
		"teams_imported":            teamsImported,
		"canned_responses_imported": cannedImported,
		"message": fmt.Sprintf(
			"Imported %d team(s) and %d canned response(s)",
			teamsImported, cannedImported,
		),
	})
}
