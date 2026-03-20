package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Email            string     `json:"email" db:"email"`
	PasswordHash     string     `json:"-" db:"password_hash"`
	FirstName        string     `json:"first_name" db:"first_name"`
	LastName         string     `json:"last_name" db:"last_name"`
	Role             string     `json:"role" db:"role"`
	Status           string     `json:"status" db:"status"`
	ParentID         *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	TwoFactorEnabled bool       `json:"two_factor_enabled" db:"two_factor_enabled"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type Server struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Hostname      string     `json:"hostname" db:"hostname"`
	IPAddress     string     `json:"ip_address" db:"ip_address"`
	IPv6Address   *string    `json:"ipv6_address,omitempty" db:"ip_v6_address"`
	Port          int        `json:"port" db:"port"`
	OS            string     `json:"os" db:"os"`
	Role          string     `json:"role" db:"role"`
	Status        string     `json:"status" db:"status"`
	AgentVersion  *string    `json:"agent_version,omitempty" db:"agent_version"`
	AgentStatus   string     `json:"agent_status" db:"agent_status"`
	SSHUser       string     `json:"ssh_user" db:"ssh_user"`
	SSHKey        string     `json:"ssh_key,omitempty" db:"ssh_key"`
	SSHPassword   string     `json:"ssh_password,omitempty" db:"ssh_password"`
	AuthMethod    string     `json:"auth_method" db:"auth_method"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty" db:"last_heartbeat"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type Domain struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID       *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	Name           string     `json:"name" db:"name"`
	Type           string     `json:"type" db:"type"`
	ParentDomainID *uuid.UUID `json:"parent_domain_id,omitempty" db:"parent_domain_id"`
	DocumentRoot   string     `json:"document_root" db:"document_root"`
	WebServer        string      `json:"web_server" db:"web_server"`
	PHPVersion       string      `json:"php_version" db:"php_version"`
	SSLEnabled       bool        `json:"ssl_enabled" db:"ssl_enabled"`
	Status           string      `json:"status" db:"status"`
	IsLoadBalancer   bool        `json:"is_load_balancer" db:"is_load_balancer"`
	BackendServerIDs []uuid.UUID `json:"backend_server_ids,omitempty" db:"-"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
}

type Application struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	DomainID     *uuid.UUID `json:"domain_id,omitempty" db:"domain_id"`
	ServerID     *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	Name         string     `json:"name" db:"name"`
	AppType      string     `json:"app_type" db:"app_type"`
	Runtime      string     `json:"runtime" db:"runtime"`
	DeployMethod string     `json:"deploy_method" db:"deploy_method"`
	GitRepo      string     `json:"git_repo,omitempty" db:"git_repo"`
	GitBranch    string     `json:"git_branch" db:"git_branch"`
	Status       string     `json:"status" db:"status"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type Task struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Type        string     `json:"type" db:"type"`
	Status      string     `json:"status" db:"status"`
	Priority    int        `json:"priority" db:"priority"`
	ServerID    *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	UserID      *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Error       string     `json:"error,omitempty" db:"error"`
	Attempts    int        `json:"attempts" db:"attempts"`
	MaxAttempts int        `json:"max_attempts" db:"max_attempts"`
	ScheduledAt time.Time  `json:"scheduled_at" db:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type AuditLog struct {
	ID         int64      `json:"id" db:"id"`
	UserID     *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Action     string     `json:"action" db:"action"`
	Resource   string     `json:"resource" db:"resource"`
	ResourceID *uuid.UUID `json:"resource_id,omitempty" db:"resource_id"`
	IPAddress  string     `json:"ip_address" db:"ip_address"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

type ServerMetrics struct {
	ID           int64     `json:"id" db:"id"`
	ServerID     uuid.UUID `json:"server_id" db:"server_id"`
	CPUPercent   float64   `json:"cpu_percent" db:"cpu_percent"`
	RAMUsedMB    int64     `json:"ram_used_mb" db:"ram_used_mb"`
	RAMTotalMB   int64     `json:"ram_total_mb" db:"ram_total_mb"`
	DiskUsedGB   float64   `json:"disk_used_gb" db:"disk_used_gb"`
	DiskTotalGB  float64   `json:"disk_total_gb" db:"disk_total_gb"`
	LoadAvg1     float64   `json:"load_avg_1" db:"load_avg_1"`
	LoadAvg5     float64   `json:"load_avg_5" db:"load_avg_5"`
	LoadAvg15    float64   `json:"load_avg_15" db:"load_avg_15"`
	NetworkRxBytes int64   `json:"network_rx_bytes" db:"network_rx_bytes"`
	NetworkTxBytes int64   `json:"network_tx_bytes" db:"network_tx_bytes"`
	RecordedAt   time.Time `json:"recorded_at" db:"recorded_at"`
}

// Phase 2 Models

type Database struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID      *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	Name          string     `json:"name" db:"name"`
	Engine        string     `json:"engine" db:"engine"`
	DBUser        string     `json:"db_user" db:"db_user"`
	DBPasswordEnc string     `json:"-" db:"db_password_enc"`
	Charset       string     `json:"charset" db:"charset"`
	SizeMB        float64    `json:"size_mb" db:"size_mb"`
	Status        string     `json:"status" db:"status"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type EmailAccount struct {
	ID           uuid.UUID `json:"id" db:"id"`
	DomainID     uuid.UUID `json:"domain_id" db:"domain_id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	Address      string    `json:"address" db:"address"`
	PasswordHash string    `json:"-" db:"password_hash"`
	QuotaMB      int       `json:"quota_mb" db:"quota_mb"`
	UsedMB       float64   `json:"used_mb" db:"used_mb"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type EmailForwarder struct {
	ID          uuid.UUID `json:"id" db:"id"`
	DomainID    uuid.UUID `json:"domain_id" db:"domain_id"`
	Source      string    `json:"source" db:"source"`
	Destination string    `json:"destination" db:"destination"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type EmailAlias struct {
	ID          uuid.UUID `json:"id" db:"id"`
	DomainID    uuid.UUID `json:"domain_id" db:"domain_id"`
	Source      string    `json:"source" db:"source"`
	Destination string    `json:"destination" db:"destination"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type EmailAutoresponder struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	AccountID uuid.UUID  `json:"account_id" db:"account_id"`
	Subject   string     `json:"subject" db:"subject"`
	Body      string     `json:"body" db:"body"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	StartDate *time.Time `json:"start_date" db:"start_date"`
	EndDate   *time.Time `json:"end_date" db:"end_date"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type EmailDNSStatus struct {
	SPF   DNSRecord `json:"spf"`
	DKIM  DNSRecord `json:"dkim"`
	DMARC DNSRecord `json:"dmarc"`
}

type DNSRecord struct {
	Found    bool   `json:"found"`
	Value    string `json:"value"`
	Expected string `json:"expected"`
}

type Backup struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID    *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	Type        string     `json:"type" db:"type"`
	Storage     string     `json:"storage" db:"storage"`
	Path        string     `json:"path,omitempty" db:"path"`
	SizeMB      float64    `json:"size_mb" db:"size_mb"`
	Status      string     `json:"status" db:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type BackupSchedule struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID      *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	Frequency     string     `json:"frequency" db:"frequency"`
	RetentionDays int        `json:"retention_days" db:"retention_days"`
	Type          string     `json:"type" db:"type"`
	Storage       string     `json:"storage" db:"storage"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt     *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// Phase 3 Models

type Deployment struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	AppID       uuid.UUID  `json:"app_id" db:"app_id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	CommitHash  string     `json:"commit_hash" db:"commit_hash"`
	Branch      string     `json:"branch" db:"branch"`
	Status      string     `json:"status" db:"status"`
	BuildLog    string     `json:"build_log,omitempty" db:"build_log"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type FirewallRule struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ServerID    uuid.UUID `json:"server_id" db:"server_id"`
	Direction   string    `json:"direction" db:"direction"`
	Protocol    string    `json:"protocol" db:"protocol"`
	Port        string    `json:"port" db:"port"`
	SourceIP    string    `json:"source_ip" db:"source_ip"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type SecurityEvent struct {
	ID        int64      `json:"id" db:"id"`
	ServerID  *uuid.UUID `json:"server_id,omitempty" db:"server_id"`
	EventType string     `json:"event_type" db:"event_type"`
	SourceIP  string     `json:"source_ip" db:"source_ip"`
	Details   string     `json:"details" db:"details"`
	Severity  string     `json:"severity" db:"severity"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type WAFConfig struct {
	ID              uuid.UUID `json:"id" db:"id"`
	ServerID        uuid.UUID `json:"server_id" db:"server_id"`
	Enabled         bool      `json:"enabled" db:"enabled"`
	Mode            string    `json:"mode" db:"mode"` // detection_only, blocking
	ParanoidLevel   int       `json:"paranoid_level" db:"paranoid_level"` // 1-4
	AllowedMethods  string    `json:"allowed_methods" db:"allowed_methods"`
	MaxRequestBody  int       `json:"max_request_body" db:"max_request_body"` // bytes
	CRSEnabled      bool      `json:"crs_enabled" db:"crs_enabled"`
	SQLiProtection  bool      `json:"sqli_protection" db:"sqli_protection"`
	XSSProtection   bool      `json:"xss_protection" db:"xss_protection"`
	RFIProtection   bool      `json:"rfi_protection" db:"rfi_protection"`
	LFIProtection   bool      `json:"lfi_protection" db:"lfi_protection"`
	RCEProtection   bool      `json:"rce_protection" db:"rce_protection"`
	ScannerBlock    bool      `json:"scanner_block" db:"scanner_block"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type WAFRule struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ServerID    uuid.UUID `json:"server_id" db:"server_id"`
	RuleID      int       `json:"rule_id" db:"rule_id"`
	Description string    `json:"description" db:"description"`
	IsDisabled  bool      `json:"is_disabled" db:"is_disabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type WAFWhitelist struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ServerID  uuid.UUID `json:"server_id" db:"server_id"`
	Type      string    `json:"type" db:"type"` // ip, uri, rule
	Value     string    `json:"value" db:"value"`
	Reason    string    `json:"reason" db:"reason"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type WAFLog struct {
	ID        int64     `json:"id" db:"id"`
	ServerID  uuid.UUID `json:"server_id" db:"server_id"`
	RuleID    int       `json:"rule_id" db:"rule_id"`
	URI       string    `json:"uri" db:"uri"`
	ClientIP  string    `json:"client_ip" db:"client_ip"`
	Message   string    `json:"message" db:"message"`
	Severity  string    `json:"severity" db:"severity"`
	Action    string    `json:"action" db:"action"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type BillingPlan struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	PriceCents   int       `json:"price_cents" db:"price_cents"`
	Currency     string    `json:"currency" db:"currency"`
	Interval     string    `json:"interval" db:"interval"`
	MaxDomains   int       `json:"max_domains" db:"max_domains"`
	MaxDatabases int       `json:"max_databases" db:"max_databases"`
	MaxEmail     int       `json:"max_email" db:"max_email"`
	DiskGB       int       `json:"disk_gb" db:"disk_gb"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Invoice struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	PlanID      *uuid.UUID `json:"plan_id,omitempty" db:"plan_id"`
	AmountCents int        `json:"amount_cents" db:"amount_cents"`
	Currency    string     `json:"currency" db:"currency"`
	Status      string     `json:"status" db:"status"`
	StripeID    string     `json:"stripe_id,omitempty" db:"stripe_id"`
	PaidAt      *time.Time `json:"paid_at,omitempty" db:"paid_at"`
	DueAt       *time.Time `json:"due_at,omitempty" db:"due_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type TransferJob struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	SourceServerID   *uuid.UUID `json:"source_server_id,omitempty" db:"source_server_id"`
	DestServerID     *uuid.UUID `json:"dest_server_id,omitempty" db:"dest_server_id"`
	SourcePath       string     `json:"source_path" db:"source_path"`
	DestPath         string     `json:"dest_path" db:"dest_path"`
	Direction        string     `json:"direction" db:"direction"`
	RsyncOptions     string     `json:"rsync_options" db:"rsync_options"`
	ExcludePatterns  string     `json:"exclude_patterns" db:"exclude_patterns"`
	BandwidthLimit   int        `json:"bandwidth_limit" db:"bandwidth_limit"`
	DeleteExtra      bool       `json:"delete_extra" db:"delete_extra"`
	DryRun           bool       `json:"dry_run" db:"dry_run"`
	Status           string     `json:"status" db:"status"`
	BytesTransferred int64      `json:"bytes_transferred" db:"bytes_transferred"`
	FilesTransferred int        `json:"files_transferred" db:"files_transferred"`
	Progress         int        `json:"progress" db:"progress"`
	Output           string     `json:"output,omitempty" db:"output"`
	StartedAt        *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

type TransferSchedule struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	Name            string     `json:"name" db:"name"`
	SourceServerID  *uuid.UUID `json:"source_server_id,omitempty" db:"source_server_id"`
	DestServerID    *uuid.UUID `json:"dest_server_id,omitempty" db:"dest_server_id"`
	SourcePath      string     `json:"source_path" db:"source_path"`
	DestPath        string     `json:"dest_path" db:"dest_path"`
	Direction       string     `json:"direction" db:"direction"`
	RsyncOptions    string     `json:"rsync_options" db:"rsync_options"`
	ExcludePatterns string     `json:"exclude_patterns" db:"exclude_patterns"`
	BandwidthLimit  int        `json:"bandwidth_limit" db:"bandwidth_limit"`
	DeleteExtra     bool       `json:"delete_extra" db:"delete_extra"`
	CronExpression  string     `json:"cron_expression" db:"cron_expression"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	LastRun         *time.Time `json:"last_run,omitempty" db:"last_run"`
	NextRun         *time.Time `json:"next_run,omitempty" db:"next_run"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}


