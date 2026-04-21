package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

// TeamService handles team member invite, accept, list, remove, and access checking
type TeamService struct {
	pool *pgxpool.Pool
}

func NewTeamService(pool *pgxpool.Pool) *TeamService {
	return &TeamService{pool: pool}
}

// InviteMember creates a pending team member record
func (s *TeamService) InviteMember(ctx context.Context, ownerID uuid.UUID, req models.InviteTeamMemberRequest) (*models.TeamMember, error) {
	// Resolve member by email
	var memberID uuid.UUID
	err := s.pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", req.Email).Scan(&memberID)
	if err != nil {
		return nil, fmt.Errorf("user with email %s not found", req.Email)
	}

	if memberID == ownerID {
		return nil, fmt.Errorf("cannot invite yourself")
	}

	role := "viewer"
	if req.Role != "" {
		role = req.Role
	}
	scopeType := "all"
	if req.ScopeType != "" {
		scopeType = req.ScopeType
	}

	var scopeID *uuid.UUID
	if req.ScopeID != "" {
		parsed, err := uuid.Parse(req.ScopeID)
		if err != nil {
			return nil, fmt.Errorf("invalid scope_id")
		}
		scopeID = &parsed
	}

	tm := &models.TeamMember{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO team_members (owner_id, member_id, role, scope_type, scope_id)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (owner_id, member_id, scope_type, scope_id) DO NOTHING
		 RETURNING id, owner_id, member_id, role, scope_type, scope_id, invited_at, accepted_at`,
		ownerID, memberID, role, scopeType, scopeID,
	).Scan(&tm.ID, &tm.OwnerID, &tm.MemberID, &tm.Role, &tm.ScopeType, &tm.ScopeID, &tm.InvitedAt, &tm.AcceptedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to invite member (may already exist): %w", err)
	}

	return tm, nil
}

// AcceptInvite marks a team member invitation as accepted
func (s *TeamService) AcceptInvite(ctx context.Context, memberID uuid.UUID, inviteID string) error {
	now := time.Now()
	result, err := s.pool.Exec(ctx,
		`UPDATE team_members SET accepted_at = $1 WHERE id = $2 AND member_id = $3 AND accepted_at IS NULL`,
		now, inviteID, memberID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("invitation not found or already accepted")
	}
	return nil
}

// ListMembers returns team members for an owner
func (s *TeamService) ListMembers(ctx context.Context, ownerID uuid.UUID) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT tm.id, tm.member_id, u.email, u.first_name, u.last_name,
		        tm.role, tm.scope_type, tm.scope_id, tm.invited_at, tm.accepted_at
		 FROM team_members tm
		 JOIN users u ON tm.member_id = u.id
		 WHERE tm.owner_id = $1
		 ORDER BY tm.invited_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var id, memberID uuid.UUID
		var email, firstName, lastName, role, scopeType string
		var scopeID *uuid.UUID
		var invitedAt time.Time
		var acceptedAt *time.Time

		if err := rows.Scan(&id, &memberID, &email, &firstName, &lastName,
			&role, &scopeType, &scopeID, &invitedAt, &acceptedAt); err != nil {
			continue
		}

		member := map[string]interface{}{
			"id":         id,
			"member_id":  memberID,
			"email":      email,
			"first_name": firstName,
			"last_name":  lastName,
			"role":       role,
			"scope_type": scopeType,
			"scope_id":   scopeID,
			"invited_at": invitedAt,
			"accepted_at": acceptedAt,
			"status":     "pending",
		}
		if acceptedAt != nil {
			member["status"] = "active"
		}
		members = append(members, member)
	}

	if members == nil {
		members = []map[string]interface{}{}
	}
	return members, nil
}

// ListPendingInvites returns pending invitations for a user
func (s *TeamService) ListPendingInvites(ctx context.Context, memberID uuid.UUID) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT tm.id, tm.owner_id, u.email, u.first_name, u.last_name,
		        tm.role, tm.scope_type, tm.scope_id, tm.invited_at
		 FROM team_members tm
		 JOIN users u ON tm.owner_id = u.id
		 WHERE tm.member_id = $1 AND tm.accepted_at IS NULL
		 ORDER BY tm.invited_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []map[string]interface{}
	for rows.Next() {
		var id, ownerID uuid.UUID
		var email, firstName, lastName, role, scopeType string
		var scopeID *uuid.UUID
		var invitedAt time.Time

		if err := rows.Scan(&id, &ownerID, &email, &firstName, &lastName,
			&role, &scopeType, &scopeID, &invitedAt); err != nil {
			continue
		}

		invites = append(invites, map[string]interface{}{
			"id":         id,
			"owner_id":   ownerID,
			"owner_email": email,
			"owner_name": firstName + " " + lastName,
			"role":       role,
			"scope_type": scopeType,
			"scope_id":   scopeID,
			"invited_at": invitedAt,
		})
	}

	if invites == nil {
		invites = []map[string]interface{}{}
	}
	return invites, nil
}

// RemoveMember deletes a team member record
func (s *TeamService) RemoveMember(ctx context.Context, ownerID uuid.UUID, memberRecordID string) error {
	result, err := s.pool.Exec(ctx,
		`DELETE FROM team_members WHERE id = $1 AND owner_id = $2`, memberRecordID, ownerID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("team member not found")
	}
	return nil
}

// CheckAccess verifies if a user has access to a specific resource through team membership
func (s *TeamService) CheckAccess(ctx context.Context, memberID uuid.UUID, scopeType string, scopeID uuid.UUID) (bool, string, error) {
	var role string
	err := s.pool.QueryRow(ctx,
		`SELECT role FROM team_members
		 WHERE member_id = $1 AND accepted_at IS NOT NULL
		 AND (scope_type = 'all' OR (scope_type = $2 AND (scope_id = $3 OR scope_id IS NULL)))
		 LIMIT 1`, memberID, scopeType, scopeID,
	).Scan(&role)
	if err != nil {
		return false, "", nil // No access
	}
	return true, role, nil
}
