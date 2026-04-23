package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
	"golang.org/x/crypto/bcrypt"
)

type EmailService struct {
	pool *pgxpool.Pool
}

func NewEmailService(pool *pgxpool.Pool) *EmailService {
	return &EmailService{pool: pool}
}

// ──────────── Accounts ────────────

func (s *EmailService) CreateAccount(ctx context.Context, userID uuid.UUID, req models.CreateEmailRequest) (*models.EmailAccount, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	quotaMB := req.QuotaMB
	if quotaMB == 0 {
		quotaMB = 1024
	}

	acct := &models.EmailAccount{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO email_accounts (domain_id, user_id, address, password_hash, quota_mb)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, domain_id, user_id, address, quota_mb, used_mb, is_active, created_at, updated_at`,
		req.DomainID, userID, req.Address, string(hash), quotaMB,
	).Scan(&acct.ID, &acct.DomainID, &acct.UserID, &acct.Address, &acct.QuotaMB, &acct.UsedMB, &acct.IsActive, &acct.CreatedAt, &acct.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create email account: %w", err)
	}

	// Provision the mailbox on the actual mail server
	go func() {
		bgCtx := context.Background()
		var serverID *uuid.UUID
		var domainName string
		if err := s.pool.QueryRow(bgCtx,
			`SELECT server_id, name FROM domains WHERE id = $1`, req.DomainID,
		).Scan(&serverID, &domainName); err != nil || serverID == nil {
			log.Printf("⚠ Email %s: no server found for domain, skipping mail provisioning", req.Address)
			return
		}
		srv, err := s.getServerSSH(bgCtx, serverID.String())
		if err != nil {
			log.Printf("⚠ Email %s: could not get server SSH info: %v", req.Address, err)
			return
		}
		localpart := strings.SplitN(req.Address, "@", 2)[0]
		script := fmt.Sprintf(`
# Postfix virtual mailbox entry
VMAILBOX=/etc/postfix/vmailbox
grep -qF '%s' "$VMAILBOX" 2>/dev/null || echo '%s %s/Maildir/' >> "$VMAILBOX"
postmap "$VMAILBOX" 2>&1

# Dovecot passwd-file entry (plain password — Dovecot hashes at auth time)
PASSWDFILE=/etc/dovecot/users
grep -qF '%s@%s' "$PASSWDFILE" 2>/dev/null || \
    echo '%s@%s:{PLAIN}%s:vmail:vmail::' >> "$PASSWDFILE" 2>&1

# Create Maildir structure
install -d -o vmail -g vmail -m 700 /var/vmail/%s/%s/Maildir/{new,cur,tmp} 2>&1 || true

# Reload services
postfix reload 2>&1 || true
doveadm reload 2>&1 || true
echo "MAIL_PROVISIONED"
`, req.Address, req.Address, domainName,
			localpart, domainName,
			localpart, domainName, req.Password,
			domainName, localpart)
		output, err := provisioner.RunScript(srv, script)
		if err != nil || !strings.Contains(output, "MAIL_PROVISIONED") {
			log.Printf("⚠ Email %s: mail server provisioning warning: %v\n%s", req.Address, err, output)
		} else {
			log.Printf("✅ Email %s: provisioned on mail server", req.Address)
		}
	}()

	return acct, nil
}

func (s *EmailService) ListAccounts(ctx context.Context, userID uuid.UUID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64

	query := `SELECT id, domain_id, user_id, address, quota_mb, used_mb, is_active, created_at, updated_at FROM email_accounts`
	countQuery := `SELECT count(*) FROM email_accounts`
	var args []interface{}

	if role != "admin" {
		args = append(args, userID)
		query += ` WHERE user_id = $1`
		countQuery += ` WHERE user_id = $1`
	}

	s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, perPage, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.EmailAccount
	for rows.Next() {
		var a models.EmailAccount
		if err := rows.Scan(&a.ID, &a.DomainID, &a.UserID, &a.Address, &a.QuotaMB, &a.UsedMB, &a.IsActive, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}

	return &models.PaginatedResponse{
		Data:       accounts,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *EmailService) checkAccountOwner(ctx context.Context, id string, userID uuid.UUID, role string) error {
	if role == "admin" {
		return nil
	}
	var ownerID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM email_accounts WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("email account not found")
	}
	if ownerID != userID {
		return fmt.Errorf("email account not found")
	}
	return nil
}

func (s *EmailService) DeleteAccount(ctx context.Context, id string, userID uuid.UUID, role string) error {
	if err := s.checkAccountOwner(ctx, id, userID, role); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM email_accounts WHERE id = $1`, id)
	return err
}

func (s *EmailService) ToggleAccount(ctx context.Context, id string, isActive bool, userID uuid.UUID, role string) error {
	if err := s.checkAccountOwner(ctx, id, userID, role); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `UPDATE email_accounts SET is_active = $1, updated_at = NOW() WHERE id = $2`, isActive, id)
	return err
}

func (s *EmailService) ChangePassword(ctx context.Context, id string, password string, userID uuid.UUID, role string) error {
	if err := s.checkAccountOwner(ctx, id, userID, role); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE email_accounts SET password_hash = $1, updated_at = NOW() WHERE id = $2`, string(hash), id)
	return err
}

func (s *EmailService) UpdateQuota(ctx context.Context, id string, quotaMB int, userID uuid.UUID, role string) error {
	if err := s.checkAccountOwner(ctx, id, userID, role); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `UPDATE email_accounts SET quota_mb = $1, updated_at = NOW() WHERE id = $2`, quotaMB, id)
	return err
}

// ──────────── Forwarders ────────────

func (s *EmailService) CreateForwarder(ctx context.Context, req models.CreateForwarderRequest) (*models.EmailForwarder, error) {
	fwd := &models.EmailForwarder{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO email_forwarders (domain_id, source, destination)
		 VALUES ($1, $2, $3)
		 RETURNING id, domain_id, source, destination, created_at`,
		req.DomainID, req.Source, req.Destination,
	).Scan(&fwd.ID, &fwd.DomainID, &fwd.Source, &fwd.Destination, &fwd.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create forwarder: %w", err)
	}
	go s.applyForwarder(req.DomainID, req.Source, req.Destination, false)
	return fwd, nil
}

func (s *EmailService) applyForwarder(domainID, source, destination string, remove bool) {
	ctx := context.Background()
	var serverID *string
	if err := s.pool.QueryRow(ctx, `SELECT server_id::text FROM domains WHERE id = $1`, domainID).Scan(&serverID); err != nil || serverID == nil {
		return
	}
	srv, err := s.getServerSSH(ctx, *serverID)
	if err != nil {
		return
	}
	var script string
	if remove {
		script = fmt.Sprintf(`
ALIAS_FILE=/etc/postfix/virtual
sed -i '/^%s /d' "$ALIAS_FILE" 2>/dev/null || true
postmap "$ALIAS_FILE" 2>/dev/null || true
postfix reload 2>/dev/null || true
echo "FORWARDER_REMOVED"`, source)
	} else {
		script = fmt.Sprintf(`
ALIAS_FILE=/etc/postfix/virtual
touch "$ALIAS_FILE"
grep -qxF '%s %s' "$ALIAS_FILE" || echo '%s %s' >> "$ALIAS_FILE"
postmap "$ALIAS_FILE" 2>/dev/null || true
postfix reload 2>/dev/null || true
echo "FORWARDER_APPLIED"`, source, destination, source, destination)
	}
	out, err := provisioner.RunScript(srv, script)
	if err != nil {
		log.Printf("⚠ Forwarder SSH error: %v — %s", err, out)
	}
}

func (s *EmailService) ListForwarders(ctx context.Context, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64
	s.pool.QueryRow(ctx, `SELECT count(*) FROM email_forwarders`).Scan(&total)

	rows, err := s.pool.Query(ctx,
		`SELECT id, domain_id, source, destination, created_at FROM email_forwarders ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.EmailForwarder
	for rows.Next() {
		var f models.EmailForwarder
		if err := rows.Scan(&f.ID, &f.DomainID, &f.Source, &f.Destination, &f.CreatedAt); err != nil {
			continue
		}
		items = append(items, f)
	}

	return &models.PaginatedResponse{
		Data: items, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *EmailService) DeleteForwarder(ctx context.Context, id string) error {
	var domainID, source, destination string
	if err := s.pool.QueryRow(ctx, `SELECT domain_id::text, source, destination FROM email_forwarders WHERE id = $1`, id).
		Scan(&domainID, &source, &destination); err == nil {
		go s.applyForwarder(domainID, source, destination, true)
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM email_forwarders WHERE id = $1`, id)
	return err
}

// ──────────── Aliases ────────────

func (s *EmailService) CreateAlias(ctx context.Context, req models.CreateAliasRequest) (*models.EmailAlias, error) {
	a := &models.EmailAlias{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO email_aliases (domain_id, source, destination)
		 VALUES ($1, $2, $3)
		 RETURNING id, domain_id, source, destination, created_at`,
		req.DomainID, req.Source, req.Destination,
	).Scan(&a.ID, &a.DomainID, &a.Source, &a.Destination, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create alias: %w", err)
	}
	go s.applyForwarder(req.DomainID, req.Source, req.Destination, false)
	return a, nil
}

func (s *EmailService) ListAliases(ctx context.Context, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64
	s.pool.QueryRow(ctx, `SELECT count(*) FROM email_aliases`).Scan(&total)

	rows, err := s.pool.Query(ctx,
		`SELECT id, domain_id, source, destination, created_at FROM email_aliases ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.EmailAlias
	for rows.Next() {
		var a models.EmailAlias
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Source, &a.Destination, &a.CreatedAt); err != nil {
			continue
		}
		items = append(items, a)
	}

	return &models.PaginatedResponse{
		Data: items, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *EmailService) DeleteAlias(ctx context.Context, id string) error {
	var domainID, source, destination string
	if err := s.pool.QueryRow(ctx, `SELECT domain_id::text, source, destination FROM email_aliases WHERE id = $1`, id).
		Scan(&domainID, &source, &destination); err == nil {
		go s.applyForwarder(domainID, source, destination, true)
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM email_aliases WHERE id = $1`, id)
	return err
}

// ──────────── Autoresponders ────────────

func (s *EmailService) CreateAutoresponder(ctx context.Context, req models.CreateAutoresponderRequest) (*models.EmailAutoresponder, error) {
	ar := &models.EmailAutoresponder{}
	var startDate, endDate *time.Time
	if req.StartDate != nil {
		t, _ := time.Parse("2006-01-02", *req.StartDate)
		startDate = &t
	}
	if req.EndDate != nil {
		t, _ := time.Parse("2006-01-02", *req.EndDate)
		endDate = &t
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO email_autoresponders (account_id, subject, body, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, account_id, subject, body, is_active, start_date, end_date, created_at`,
		req.AccountID, req.Subject, req.Body, startDate, endDate,
	).Scan(&ar.ID, &ar.AccountID, &ar.Subject, &ar.Body, &ar.IsActive, &ar.StartDate, &ar.EndDate, &ar.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create autoresponder: %w", err)
	}
	return ar, nil
}

func (s *EmailService) ListAutoresponders(ctx context.Context, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64
	s.pool.QueryRow(ctx, `SELECT count(*) FROM email_autoresponders`).Scan(&total)

	rows, err := s.pool.Query(ctx,
		`SELECT id, account_id, subject, body, is_active, start_date, end_date, created_at FROM email_autoresponders ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.EmailAutoresponder
	for rows.Next() {
		var a models.EmailAutoresponder
		if err := rows.Scan(&a.ID, &a.AccountID, &a.Subject, &a.Body, &a.IsActive, &a.StartDate, &a.EndDate, &a.CreatedAt); err != nil {
			continue
		}
		items = append(items, a)
	}

	return &models.PaginatedResponse{
		Data: items, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *EmailService) DeleteAutoresponder(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM email_autoresponders WHERE id = $1`, id)
	return err
}

func (s *EmailService) ToggleAutoresponder(ctx context.Context, id string, isActive bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE email_autoresponders SET is_active = $1 WHERE id = $2`, isActive, id)
	return err
}

// ──────────── DNS / Auth ────────────

func (s *EmailService) GetDNSStatus(ctx context.Context, domainName string) (*models.EmailDNSStatus, error) {
	status := &models.EmailDNSStatus{}

	// Check SPF
	status.SPF.Expected = fmt.Sprintf("v=spf1 a mx ~all")
	txtRecords, _ := net.LookupTXT(domainName)
	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=spf1") {
			status.SPF.Found = true
			status.SPF.Value = txt
			break
		}
	}

	// Check DKIM
	dkimHost := "default._domainkey." + domainName
	status.DKIM.Expected = "v=DKIM1; k=rsa; p=..."
	dkimRecords, _ := net.LookupTXT(dkimHost)
	for _, txt := range dkimRecords {
		if strings.Contains(txt, "DKIM1") {
			status.DKIM.Found = true
			status.DKIM.Value = txt
			break
		}
	}

	// Check DMARC
	dmarcHost := "_dmarc." + domainName
	status.DMARC.Expected = fmt.Sprintf("v=DMARC1; p=quarantine; rua=mailto:dmarc@%s", domainName)
	dmarcRecords, _ := net.LookupTXT(dmarcHost)
	for _, txt := range dmarcRecords {
		if strings.HasPrefix(txt, "v=DMARC1") {
			status.DMARC.Found = true
			status.DMARC.Value = txt
			break
		}
	}

	return status, nil
}

// ──────────── Catch-All ────────────

func (s *EmailService) SetCatchAll(ctx context.Context, domainID string, address string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE domains SET catch_all_address = $1 WHERE id = $2`, address, domainID)
	return err
}

func (s *EmailService) GetCatchAll(ctx context.Context, domainID string) (string, error) {
	var addr string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(catch_all_address, '') FROM domains WHERE id = $1`, domainID).Scan(&addr)
	return addr, err
}

// ──────────── Webmail ────────────

func (s *EmailService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	return GetServerInfo(ctx, s.pool, serverID)
}

// DeployWebmail deploys Roundcube webmail via Docker on a server
func (s *EmailService) DeployWebmail(ctx context.Context, serverID string) (map[string]string, error) {
	srv, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	cmd := `docker rm -f novapanel-roundcube 2>/dev/null || true
docker run -d --name novapanel-roundcube --restart unless-stopped \
  -e ROUNDCUBEMAIL_DEFAULT_HOST=ssl://localhost \
  -e ROUNDCUBEMAIL_DEFAULT_PORT=993 \
  -e ROUNDCUBEMAIL_SMTP_SERVER=localhost \
  -e ROUNDCUBEMAIL_SMTP_PORT=587 \
  -e ROUNDCUBEMAIL_DB_TYPE=sqlite \
  -p 8085:80 \
  --add-host=localhost:host-gateway \
  --add-host=host.docker.internal:host-gateway \
  roundcube/roundcubemail:latest 2>&1
echo "Roundcube deployed"
docker port novapanel-roundcube 80`

	output, err := provisioner.RunScript(srv, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy Roundcube: %w\nOutput: %s", err, output)
	}

	return map[string]string{
		"tool":      "Roundcube",
		"port":      "8085",
		"url":       fmt.Sprintf("http://%s:8085", srv.IPAddress),
		"server_ip": srv.IPAddress,
		"output":    strings.TrimSpace(output),
	}, nil
}

// WebmailStatus checks if the Roundcube container is running on a server
func (s *EmailService) WebmailStatus(ctx context.Context, serverID string) (map[string]string, error) {
	srv, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	cmd := `docker inspect -f '{{.State.Status}}' novapanel-roundcube 2>/dev/null || echo 'not_found'`
	output, _ := provisioner.RunScript(srv, cmd)
	status := strings.TrimSpace(output)

	info := map[string]string{
		"status":    status,
		"tool":      "Roundcube",
		"port":      "8085",
		"server_ip": srv.IPAddress,
	}
	if status == "running" {
		info["url"] = fmt.Sprintf("http://%s:8085", srv.IPAddress)
	}
	return info, nil
}

// StopWebmail stops the Roundcube container
func (s *EmailService) StopWebmail(ctx context.Context, serverID string) error {
	srv, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return err
	}
	_, err = provisioner.RunScript(srv, `docker rm -f novapanel-roundcube 2>/dev/null && echo 'stopped'`)
	return err
}
