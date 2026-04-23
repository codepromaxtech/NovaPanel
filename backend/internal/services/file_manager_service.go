package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/provisioner"
)


// FileManagerService handles SSH-based remote file management
type FileManagerService struct {
	pool *pgxpool.Pool
}

func NewFileManagerService(pool *pgxpool.Pool) *FileManagerService {
	return &FileManagerService{pool: pool}
}

func (s *FileManagerService) getServer(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var server provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err := s.pool.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &encKey, &encPassword, &server.AuthMethod)
	if err != nil {
		return server, err
	}
	server.Port = port
	if cryptoKey, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
		if encKey != "" {
			if dec, derr := novacrypto.Decrypt(encKey, cryptoKey); derr == nil {
				encKey = dec
			}
		}
		if encPassword != "" {
			if dec, derr := novacrypto.Decrypt(encPassword, cryptoKey); derr == nil {
				encPassword = dec
			}
		}
	}
	server.SSHKey = encKey
	server.SSHPassword = encPassword
	return server, nil
}

// blockedPaths are paths that client-role users must not be able to access.
var blockedPaths = []string{
	"/etc/shadow", "/etc/sudoers", "/etc/sudoers.d",
	"/root", "/proc", "/sys", "/dev",
	"/etc/ssh", "/boot", "/run/secrets",
}

// isSafePath rejects absolute paths that traverse into system-critical locations.
func isSafePath(path string) bool {
	if path == "" || path == "/" {
		return true
	}
	clean := strings.TrimRight(path, "/")
	for _, blocked := range blockedPaths {
		if clean == blocked || strings.HasPrefix(clean+"/", blocked+"/") {
			return false
		}
	}
	return true
}

// ListFiles lists files/dirs in a directory on a remote server
func (s *FileManagerService) ListFiles(ctx context.Context, serverID, path string) (string, error) {
	if path == "" {
		path = "/"
	}
	if !isSafePath(path) {
		return "", fmt.Errorf("access to this path is restricted")
	}
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	// Output JSON array with file info: name, path, is_dir, size, permissions, modified_at, owner, group
	script := fmt.Sprintf(`python3 -c "
import os, json, stat, pwd, grp, time
path = '%s'
if not os.path.isdir(path):
    print(json.dumps([]))
    exit(0)
items = []
for name in sorted(os.listdir(path)):
    fp = os.path.join(path, name)
    try:
        st = os.lstat(fp)
        items.append({
            'name': name,
            'path': fp,
            'is_dir': stat.S_ISDIR(st.st_mode),
            'is_link': stat.S_ISLNK(st.st_mode),
            'size': st.st_size,
            'permissions': stat.filemode(st.st_mode),
            'modified_at': time.strftime('%%Y-%%m-%%d %%H:%%M:%%S', time.localtime(st.st_mtime)),
            'owner': pwd.getpwuid(st.st_uid).pw_name if st.st_uid < 65534 else str(st.st_uid),
            'group': grp.getgrgid(st.st_gid).gr_name if st.st_gid < 65534 else str(st.st_gid),
        })
    except: pass
print(json.dumps(items))
" 2>/dev/null || ls -la --time-style=long-iso '%s' | tail -n +2 | awk '{printf "%%s %%s %%s %%s %%s\n", $1, $3, $4, $5, $NF}'`, path, path)
	return provisioner.RunScript(server, script)
}

// ReadFile reads a file content from remote server (base64 for binary safety)
func (s *FileManagerService) ReadFile(ctx context.Context, serverID, path string) (string, error) {
	if !isSafePath(path) {
		return "", fmt.Errorf("access to this path is restricted")
	}
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`#!/bin/bash
FILE='%s'
if [ ! -f "$FILE" ]; then echo "ERROR:File not found"; exit 1; fi
SIZE=$(stat -c %%s "$FILE" 2>/dev/null || stat -f %%z "$FILE" 2>/dev/null)
if [ "$SIZE" -gt 10485760 ]; then echo "ERROR:File too large (>10MB)"; exit 1; fi
# Check if binary
if file -b --mime-encoding "$FILE" 2>/dev/null | grep -q "binary"; then
    echo "BINARY:$(base64 -w0 "$FILE")"
else
    cat "$FILE"
fi
`, path)
	return provisioner.RunScript(server, script)
}

// WriteFile writes content to a file on remote server
func (s *FileManagerService) WriteFile(ctx context.Context, serverID, path, content string) (string, error) {
	if !isSafePath(path) {
		return "", fmt.Errorf("access to this path is restricted")
	}
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	script := fmt.Sprintf(`echo '%s' | base64 -d > '%s' && echo "File saved successfully"`, encoded, path)
	return provisioner.RunScript(server, script)
}

// CreateFile creates a new empty file
func (s *FileManagerService) CreateFile(ctx context.Context, serverID, path string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`touch '%s' && echo "File created"`, path)
	return provisioner.RunScript(server, script)
}

// CreateDir creates a new directory
func (s *FileManagerService) CreateDir(ctx context.Context, serverID, path string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`mkdir -p '%s' && echo "Directory created"`, path)
	return provisioner.RunScript(server, script)
}

// Rename renames a file or directory
func (s *FileManagerService) Rename(ctx context.Context, serverID, oldPath, newPath string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`mv '%s' '%s' && echo "Renamed successfully"`, oldPath, newPath)
	return provisioner.RunScript(server, script)
}

// Copy copies a file or directory
func (s *FileManagerService) Copy(ctx context.Context, serverID, src, dest string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`cp -rp '%s' '%s' && echo "Copied successfully"`, src, dest)
	return provisioner.RunScript(server, script)
}

// Move moves a file or directory
func (s *FileManagerService) Move(ctx context.Context, serverID, src, dest string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`mv '%s' '%s' && echo "Moved successfully"`, src, dest)
	return provisioner.RunScript(server, script)
}

// Delete deletes a file or directory
func (s *FileManagerService) Delete(ctx context.Context, serverID, path string) (string, error) {
	if !isSafePath(path) {
		return "", fmt.Errorf("access to this path is restricted")
	}
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`rm -rf '%s' && echo "Deleted successfully"`, path)
	return provisioner.RunScript(server, script)
}

// Chmod changes file permissions
func (s *FileManagerService) Chmod(ctx context.Context, serverID, path, mode string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`chmod %s '%s' && echo "Permissions changed"`, mode, path)
	return provisioner.RunScript(server, script)
}

// Chown changes file ownership
func (s *FileManagerService) Chown(ctx context.Context, serverID, path, owner string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`chown -R %s '%s' && echo "Ownership changed"`, owner, path)
	return provisioner.RunScript(server, script)
}

// Search searches for files matching a pattern
func (s *FileManagerService) Search(ctx context.Context, serverID, path, query string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`find '%s' -maxdepth 5 -name '*%s*' 2>/dev/null | head -100`, path, query)
	return provisioner.RunScript(server, script)
}

// FileInfo gets detailed info about a single file
func (s *FileManagerService) FileInfo(ctx context.Context, serverID, path string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`python3 -c "
import os, json, stat, pwd, grp, time
fp = '%s'
st = os.lstat(fp)
info = {
    'name': os.path.basename(fp),
    'path': fp,
    'is_dir': stat.S_ISDIR(st.st_mode),
    'is_link': stat.S_ISLNK(st.st_mode),
    'size': st.st_size,
    'permissions': stat.filemode(st.st_mode),
    'permissions_octal': oct(stat.S_IMODE(st.st_mode)),
    'modified_at': time.strftime('%%Y-%%m-%%d %%H:%%M:%%S', time.localtime(st.st_mtime)),
    'created_at': time.strftime('%%Y-%%m-%%d %%H:%%M:%%S', time.localtime(st.st_ctime)),
    'owner': pwd.getpwuid(st.st_uid).pw_name,
    'group': grp.getgrgid(st.st_gid).gr_name,
    'inode': st.st_ino,
}
if stat.S_ISLNK(st.st_mode):
    info['link_target'] = os.readlink(fp)
print(json.dumps(info))
" 2>/dev/null || stat '%s'`, path, path)
	return provisioner.RunScript(server, script)
}

// DiskUsage gets disk usage for a path
func (s *FileManagerService) DiskUsage(ctx context.Context, serverID, path string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`du -sh '%s' 2>/dev/null && df -h '%s' 2>/dev/null | tail -1`, path, path)
	return provisioner.RunScript(server, script)
}

// Extract extracts archives (tar, zip, gz)
func (s *FileManagerService) Extract(ctx context.Context, serverID, path, dest string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if dest == "" {
		dest = "$(dirname '" + path + "')"
	}
	script := fmt.Sprintf(`#!/bin/bash
FILE='%s'
DEST='%s'
EXT="${FILE##*.}"
case "$EXT" in
    zip) unzip -o "$FILE" -d "$DEST" ;;
    gz|tgz) tar xzf "$FILE" -C "$DEST" ;;
    bz2) tar xjf "$FILE" -C "$DEST" ;;
    xz) tar xJf "$FILE" -C "$DEST" ;;
    tar) tar xf "$FILE" -C "$DEST" ;;
    *) echo "Unsupported format: $EXT"; exit 1 ;;
esac
echo "Extracted successfully"`, path, dest)
	return provisioner.RunScript(server, script)
}

// Compress creates a tar.gz archive
func (s *FileManagerService) Compress(ctx context.Context, serverID, path, dest string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if dest == "" {
		dest = path + ".tar.gz"
	}
	script := fmt.Sprintf(`tar czf '%s' -C "$(dirname '%s')" "$(basename '%s')" && echo "Compressed to %s"`, dest, path, path, dest)
	return provisioner.RunScript(server, script)
}

// GrepInFiles searches content inside files
func (s *FileManagerService) GrepInFiles(ctx context.Context, serverID, path, pattern string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`grep -rn --include='*.{php,html,css,js,py,go,json,xml,yml,yaml,conf,sh,txt,md,env,log}' '%s' '%s' 2>/dev/null | head -200`, pattern, path)
	return provisioner.RunScript(server, script)
}

// Helper to parse JSON output or return raw
func ParseFileList(output string) ([]map[string]interface{}, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return []map[string]interface{}{}, nil
	}
	var files []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &files); err != nil {
		// fallback: return raw as single item
		return []map[string]interface{}{{"raw": output}}, nil
	}
	return files, nil
}
