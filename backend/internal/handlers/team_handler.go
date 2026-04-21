package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type TeamHandler struct {
	service *services.TeamService
}

func NewTeamHandler(service *services.TeamService) *TeamHandler {
	return &TeamHandler{service: service}
}

// POST /api/v1/team/invite
func (h *TeamHandler) Invite(c *gin.Context) {
	var req models.InviteTeamMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	ownerID, _ := c.Get("user_id")
	uid, err := uuid.Parse(ownerID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	tm, err := h.service.InviteMember(c.Request.Context(), uid, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tm)
}

// POST /api/v1/team/accept/:id
func (h *TeamHandler) Accept(c *gin.Context) {
	inviteID := c.Param("id")
	memberIDStr, _ := c.Get("user_id")
	memberID, err := uuid.Parse(memberIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	if err := h.service.AcceptInvite(c.Request.Context(), memberID, inviteID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Invitation accepted"})
}

// GET /api/v1/team/members
func (h *TeamHandler) ListMembers(c *gin.Context) {
	ownerIDStr, _ := c.Get("user_id")
	ownerID, err := uuid.Parse(ownerIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	members, err := h.service.ListMembers(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": members})
}

// GET /api/v1/team/invites
func (h *TeamHandler) ListInvites(c *gin.Context) {
	memberIDStr, _ := c.Get("user_id")
	memberID, err := uuid.Parse(memberIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	invites, err := h.service.ListPendingInvites(c.Request.Context(), memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": invites})
}

// DELETE /api/v1/team/members/:id
func (h *TeamHandler) Remove(c *gin.Context) {
	memberRecordID := c.Param("id")
	ownerIDStr, _ := c.Get("user_id")
	ownerID, err := uuid.Parse(ownerIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), ownerID, memberRecordID); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Team member removed"})
}
