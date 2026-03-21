package models

// Request/Response DTOs

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      User   `json:"user"`
}

type CreateDomainRequest struct {
	Name             string   `json:"name" binding:"required"`
	Type             string   `json:"type"`
	ServerID         string   `json:"server_id"`
	WebServer        string   `json:"web_server"`
	PHPVersion       string   `json:"php_version"`
	DocumentRoot     string   `json:"document_root"`
	IsLoadBalancer   bool     `json:"is_load_balancer"`
	BackendServerIDs []string `json:"backend_server_ids"`
}

type UpdateDomainRequest struct {
	WebServer    string `json:"web_server"`
	PHPVersion   string `json:"php_version"`
	DocumentRoot string `json:"document_root"`
	SSLEnabled   *bool  `json:"ssl_enabled"`
	Status       string `json:"status"`
}

type CreateServerRequest struct {
	Name        string   `json:"name" binding:"required"`
	Hostname    string   `json:"hostname" binding:"required"`
	IPAddress   string   `json:"ip_address" binding:"required"`
	Port        int      `json:"port"`
	OS          string   `json:"os"`
	Role        string   `json:"role"`
	SSHUser     string   `json:"ssh_user"`
	SSHKey      string   `json:"ssh_key"`
	SSHPassword string   `json:"ssh_password"`
	AuthMethod  string   `json:"auth_method"`
	Modules     []string `json:"modules"` // modules to auto-install
}

type UpdateServerRequest struct {
	Name        string `json:"name"`
	Hostname    string `json:"hostname"`
	IPAddress   string `json:"ip_address"`
	Port        int    `json:"port"`
	OS          string `json:"os"`
	Role        string `json:"role"`
	SSHUser     string `json:"ssh_user"`
	SSHKey      string `json:"ssh_key"`
	SSHPassword string `json:"ssh_password"`
	AuthMethod  string `json:"auth_method"`
}

type TestConnectionRequest struct {
	IPAddress   string `json:"ip_address" binding:"required"`
	Port        int    `json:"port"`
	SSHUser     string `json:"ssh_user"`
	SSHKey      string `json:"ssh_key"`
	SSHPassword string `json:"ssh_password"`
	AuthMethod  string `json:"auth_method"`
}

type CreateTaskRequest struct {
	Type     string      `json:"type" binding:"required"`
	Payload  interface{} `json:"payload"`
	Priority int         `json:"priority"`
	ServerID string      `json:"server_id"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type DashboardStats struct {
	TotalServers  int64 `json:"total_servers"`
	ActiveServers int64 `json:"active_servers"`
	TotalDomains  int64 `json:"total_domains"`
	TotalUsers    int64 `json:"total_users"`
	TotalApps     int64 `json:"total_apps"`
	PendingTasks  int64 `json:"pending_tasks"`
}

// Phase 2 DTOs

type CreateDatabaseRequest struct {
	Name     string `json:"name" binding:"required"`
	Engine   string `json:"engine"`
	ServerID string `json:"server_id"`
	Charset  string `json:"charset"`
}

type CreateEmailRequest struct {
	DomainID string `json:"domain_id" binding:"required"`
	Address  string `json:"address" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	QuotaMB  int    `json:"quota_mb"`
}

type CreateForwarderRequest struct {
	DomainID    string `json:"domain_id" binding:"required"`
	Source      string `json:"source" binding:"required"`
	Destination string `json:"destination" binding:"required"`
}

type CreateAliasRequest struct {
	DomainID    string `json:"domain_id" binding:"required"`
	Source      string `json:"source" binding:"required"`
	Destination string `json:"destination" binding:"required"`
}

type CreateAutoresponderRequest struct {
	AccountID string  `json:"account_id" binding:"required"`
	Subject   string  `json:"subject" binding:"required"`
	Body      string  `json:"body" binding:"required"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

type ChangePasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

type UpdateQuotaRequest struct {
	QuotaMB int `json:"quota_mb" binding:"required,min=1"`
}

type ToggleRequest struct {
	IsActive bool `json:"is_active"`
}

type SetCatchAllRequest struct {
	Address string `json:"address"`
}

type CreateBackupRequest struct {
	ServerID string `json:"server_id"`
	Type     string `json:"type"`
	Storage  string `json:"storage"`
}

type CreateBackupScheduleRequest struct {
	ServerID      string `json:"server_id"`
	Frequency     string `json:"frequency" binding:"required"`
	RetentionDays int    `json:"retention_days"`
	Type          string `json:"type"`
	Storage       string `json:"storage"`
}

type FileListRequest struct {
	Path     string `json:"path" binding:"required"`
	ServerID string `json:"server_id"`
}

type FileEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	Permissions string `json:"permissions"`
	ModifiedAt  string `json:"modified_at"`
}

type FileContentResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// Phase 3 DTOs

type CreateDeploymentRequest struct {
	AppID  string `json:"app_id" binding:"required"`
	Branch string `json:"branch"`
}

type CreateFirewallRuleRequest struct {
	ServerID    string `json:"server_id" binding:"required"`
	Direction   string `json:"direction"`
	Protocol    string `json:"protocol"`
	Port        string `json:"port" binding:"required"`
	SourceIP    string `json:"source_ip"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type CreateBillingPlanRequest struct {
	Name         string `json:"name" binding:"required"`
	PriceCents   int    `json:"price_cents" binding:"required"`
	Currency     string `json:"currency"`
	Interval     string `json:"interval"`
	MaxDomains   int    `json:"max_domains"`
	MaxDatabases int    `json:"max_databases"`
	MaxEmail     int    `json:"max_email"`
	DiskGB       int    `json:"disk_gb"`
}

type UpdateProfileRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateWAFConfigRequest struct {
	Enabled        *bool   `json:"enabled"`
	Mode           string  `json:"mode"`
	ParanoidLevel  *int    `json:"paranoid_level"`
	AllowedMethods string  `json:"allowed_methods"`
	MaxRequestBody *int    `json:"max_request_body"`
	CRSEnabled     *bool   `json:"crs_enabled"`
	SQLiProtection *bool   `json:"sqli_protection"`
	XSSProtection  *bool   `json:"xss_protection"`
	RFIProtection  *bool   `json:"rfi_protection"`
	LFIProtection  *bool   `json:"lfi_protection"`
	RCEProtection  *bool   `json:"rce_protection"`
	ScannerBlock   *bool   `json:"scanner_block"`
}

type DisableWAFRuleRequest struct {
	RuleID      int    `json:"rule_id" binding:"required"`
	Description string `json:"description"`
}

type CreateWAFWhitelistRequest struct {
	Type   string `json:"type" binding:"required"` // ip, uri, rule
	Value  string `json:"value" binding:"required"`
	Reason string `json:"reason"`
}

type CreateTransferRequest struct {
	SourceServerID  string `json:"source_server_id"`
	DestServerID    string `json:"dest_server_id"`
	SourcePath      string `json:"source_path" binding:"required"`
	DestPath        string `json:"dest_path" binding:"required"`
	Direction       string `json:"direction" binding:"required"` // push, pull, local
	RsyncOptions    string `json:"rsync_options"`
	ExcludePatterns string `json:"exclude_patterns"`
	BandwidthLimit  int    `json:"bandwidth_limit"`
	DeleteExtra     bool   `json:"delete_extra"`
	DryRun          bool   `json:"dry_run"`
}

type CreateScheduleRequest struct {
	Name            string `json:"name" binding:"required"`
	SourceServerID  string `json:"source_server_id"`
	DestServerID    string `json:"dest_server_id"`
	SourcePath      string `json:"source_path" binding:"required"`
	DestPath        string `json:"dest_path" binding:"required"`
	Direction       string `json:"direction" binding:"required"`
	RsyncOptions    string `json:"rsync_options"`
	ExcludePatterns string `json:"exclude_patterns"`
	BandwidthLimit  int    `json:"bandwidth_limit"`
	DeleteExtra     bool   `json:"delete_extra"`
	CronExpression  string `json:"cron_expression" binding:"required"`
}
