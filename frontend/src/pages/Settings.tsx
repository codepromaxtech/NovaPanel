import { useState, useEffect } from 'react';
import { Settings2, User, Lock, Bell, Save, Loader, Key, Shield, Monitor, Plus, Trash2, Copy, Check, QrCode, X, LogOut, Mail, CreditCard, Send, RefreshCw, Download, ExternalLink, AlertTriangle, CheckCircle2, Zap } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import { settingsService } from '../services/billing';
import { authService } from '../services/auth';
import { updateService } from '../services/system';
import { useToast } from '../components/ui/ToastProvider';
import { useModalLock } from '../hooks/useModalLock';

const inputCls = 'w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20';
const btnPrimary = 'flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-50';

export default function Settings() {
    const toast = useToast();
    const [searchParams] = useSearchParams();
    const [activeTab, setActiveTab] = useState(searchParams.get('tab') || 'profile');
    const [profile, setProfile] = useState({ name: '', email: '' });
    const [passwords, setPasswords] = useState({ current: '', new_pass: '', confirm: '' });
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);

    // API Keys
    const [apiKeys, setApiKeys] = useState<any[]>([]);
    const [keysLoading, setKeysLoading] = useState(false);
    const [showNewKey, setShowNewKey] = useState(false);
    const [newKeyName, setNewKeyName] = useState('');
    const [newKeyScopes, setNewKeyScopes] = useState<string[]>(['read']);
    const [createdKey, setCreatedKey] = useState<string | null>(null);
    const [keyCopied, setKeyCopied] = useState(false);
    useModalLock(showNewKey || !!createdKey);

    // 2FA
    const [totpSetup, setTotpSetup] = useState<{ secret: string; qr_uri: string } | null>(null);
    const [totpCode, setTotpCode] = useState('');
    const [backupCodes, setBackupCodes] = useState<string[]>([]);
    const [totpLoading, setTotpLoading] = useState(false);
    const [disablePassword, setDisablePassword] = useState('');
    useModalLock(!!totpSetup || backupCodes.length > 0);

    // Sessions
    const [sessions, setSessions] = useState<any[]>([]);
    const [sessionsLoading, setSessionsLoading] = useState(false);

    // System settings (admin)
    const [userRole, setUserRole] = useState('');
    const [smtpSettings, setSmtpSettings] = useState({ smtp_host: '', smtp_port: '587', smtp_user: '', smtp_password: '', smtp_from: '' });
    const [stripeSettings, setStripeSettings] = useState({ stripe_secret_key: '', stripe_webhook_secret: '', stripe_price_enterprise: '', stripe_price_reseller: '' });
    const [systemLoading, setSystemLoading] = useState(false);
    const [testingSmtp, setTestingSmtp] = useState(false);

    // Updates (admin)
    const [updateStatus, setUpdateStatus] = useState<{
        current_version: string; latest_version: string; update_available: boolean;
        release_notes: string; release_url: string; last_checked: string; applying: boolean;
    } | null>(null);
    const [updateLoading, setUpdateLoading] = useState(false);
    const [checking, setChecking] = useState(false);
    const [applyLog, setApplyLog] = useState<string[]>([]);
    const [updateWs, setUpdateWs] = useState<WebSocket | null>(null);

    useEffect(() => {
        settingsService.getProfile()
            .then(data => {
                setProfile({ name: data.name || '', email: data.email || '' });
                setUserRole(data.role || '');
            })
            .catch(() => setProfile({ name: 'Admin User', email: 'admin@novapanel.io' }))
            .finally(() => setLoading(false));
    }, []);

    useEffect(() => {
        if (activeTab === 'api-keys') loadApiKeys();
        if (activeTab === 'sessions') loadSessions();
        if (activeTab === 'smtp' || activeTab === 'stripe') loadSystemSettings();
        if (activeTab === 'updates') loadUpdateStatus();
    }, [activeTab]);

    const loadApiKeys = async () => {
        setKeysLoading(true);
        try { setApiKeys(await settingsService.listApiKeys()); } catch { setApiKeys([]); }
        finally { setKeysLoading(false); }
    };

    const loadSessions = async () => {
        setSessionsLoading(true);
        try { setSessions(await authService.listSessions()); } catch { setSessions([]); }
        finally { setSessionsLoading(false); }
    };

    const handleSaveProfile = async () => {
        setSaving(true);
        try {
            await settingsService.updateProfile(profile);
            toast.success('Profile updated');
        } catch { toast.error('Failed to update profile'); }
        finally { setSaving(false); }
    };

    const handleChangePassword = async () => {
        if (passwords.new_pass !== passwords.confirm) { toast.error('Passwords do not match'); return; }
        setSaving(true);
        try {
            await settingsService.updateProfile({ password: passwords.new_pass });
            toast.success('Password changed');
            setPasswords({ current: '', new_pass: '', confirm: '' });
        } catch { toast.error('Failed to change password'); }
        finally { setSaving(false); }
    };

    const handleCreateKey = async () => {
        if (!newKeyName.trim()) return;
        setSaving(true);
        try {
            const result = await settingsService.createApiKey(newKeyName, newKeyScopes);
            setCreatedKey(result.raw_key);
            setShowNewKey(false);
            setNewKeyName('');
            await loadApiKeys();
        } catch { toast.error('Failed to create API key'); }
        finally { setSaving(false); }
    };

    const handleRevokeKey = async (id: string) => {
        try {
            await settingsService.revokeApiKey(id);
            toast.success('API key revoked');
            await loadApiKeys();
        } catch { toast.error('Failed to revoke key'); }
    };

    const handleCopyKey = () => {
        if (!createdKey) return;
        navigator.clipboard.writeText(createdKey);
        setKeyCopied(true);
        setTimeout(() => setKeyCopied(false), 2000);
    };

    const handleTotpSetup = async () => {
        setTotpLoading(true);
        try { setTotpSetup(await authService.totpSetup()); }
        catch { toast.error('Failed to start 2FA setup'); }
        finally { setTotpLoading(false); }
    };

    const handleTotpEnable = async () => {
        if (!totpCode || totpCode.length < 6) return;
        setTotpLoading(true);
        try {
            const res = await authService.totpEnable(totpCode);
            setBackupCodes(res.backup_codes);
            setTotpSetup(null);
            setTotpCode('');
            toast.success('Two-factor authentication enabled');
        } catch { toast.error('Invalid code — try again'); }
        finally { setTotpLoading(false); }
    };

    const handleTotpDisable = async () => {
        if (!disablePassword) return;
        setTotpLoading(true);
        try {
            await authService.totpDisable(disablePassword);
            setDisablePassword('');
            toast.success('Two-factor authentication disabled');
        } catch { toast.error('Invalid password'); }
        finally { setTotpLoading(false); }
    };

    const handleRevokeSession = async (id: string) => {
        try {
            await authService.revokeSession(id);
            toast.success('Session revoked');
            setSessions(s => s.filter(x => x.id !== id));
        } catch { toast.error('Failed to revoke session'); }
    };

    const handleRevokeAll = async () => {
        try {
            await authService.revokeAllOtherSessions();
            toast.success('All other sessions revoked');
            await loadSessions();
        } catch { toast.error('Failed to revoke sessions'); }
    };

    const loadSystemSettings = async () => {
        setSystemLoading(true);
        try {
            const data = await settingsService.getSystemSettings();
            setSmtpSettings({
                smtp_host: data.smtp_host || '',
                smtp_port: data.smtp_port || '587',
                smtp_user: data.smtp_user || '',
                smtp_password: data.smtp_password || '',
                smtp_from: data.smtp_from || '',
            });
            setStripeSettings({
                stripe_secret_key: data.stripe_secret_key || '',
                stripe_webhook_secret: data.stripe_webhook_secret || '',
                stripe_price_enterprise: data.stripe_price_enterprise || '',
                stripe_price_reseller: data.stripe_price_reseller || '',
            });
        } catch { toast.error('Failed to load system settings'); }
        finally { setSystemLoading(false); }
    };

    const handleSaveSmtp = async () => {
        setSaving(true);
        try {
            await settingsService.updateSystemSettings(smtpSettings);
            toast.success('SMTP settings saved');
        } catch { toast.error('Failed to save SMTP settings'); }
        finally { setSaving(false); }
    };

    const handleTestSmtp = async () => {
        setTestingSmtp(true);
        try {
            await settingsService.testSmtp();
            toast.success('Test email sent — check your inbox');
        } catch (e: any) {
            toast.error(e?.message || 'SMTP test failed');
        } finally { setTestingSmtp(false); }
    };

    const handleSaveStripe = async () => {
        setSaving(true);
        try {
            await settingsService.updateSystemSettings(stripeSettings);
            toast.success('Stripe settings saved');
        } catch { toast.error('Failed to save Stripe settings'); }
        finally { setSaving(false); }
    };

    const loadUpdateStatus = async () => {
        setUpdateLoading(true);
        try { setUpdateStatus(await updateService.getStatus()); } catch { /* not critical */ }
        finally { setUpdateLoading(false); }
    };

    const handleCheckUpdates = async () => {
        setChecking(true);
        try {
            const status = await updateService.checkNow();
            setUpdateStatus(status);
            if (status.update_available) toast.success('New version available: ' + status.latest_version);
            else toast.success('NovaPanel is up to date');
        } catch { toast.error('Failed to check for updates'); }
        finally { setChecking(false); }
    };

    const handleApplyUpdate = async () => {
        setApplyLog(['Starting update...']);
        try {
            await updateService.applyUpdate();
            // Connect WebSocket to stream progress
            const wsUrl = (window.location.protocol === 'https:' ? 'wss' : 'ws') + '://' + window.location.host + '/api/v1/ws';
            const ws = new WebSocket(wsUrl);
            setUpdateWs(ws);
            ws.onmessage = (e) => {
                try {
                    const event = JSON.parse(e.data);
                    if (event.type === 'update:progress') {
                        setApplyLog(l => [...l, `[${event.payload.percent}%] ${event.payload.message}`]);
                    } else if (event.type === 'update:complete') {
                        setApplyLog(l => [...l, '✓ Update applied! Reconnecting in 10 seconds...']);
                        ws.close();
                        setTimeout(() => window.location.reload(), 10000);
                    } else if (event.type === 'update:error') {
                        setApplyLog(l => [...l, '✖ Error: ' + event.payload.message]);
                        toast.error(event.payload.message);
                    }
                } catch { /* ignore parse errors */ }
            };
        } catch (e: any) { toast.error(e?.response?.data?.error || 'Failed to start update'); }
    };

    const isAdmin = userRole === 'admin';
    const tabs = [
        { id: 'profile', label: 'Profile', icon: User },
        { id: 'security', label: 'Password', icon: Lock },
        { id: '2fa', label: 'Two-Factor Auth', icon: Shield },
        { id: 'api-keys', label: 'API Keys', icon: Key },
        { id: 'sessions', label: 'Sessions', icon: Monitor },
        { id: 'notifications', label: 'Notifications', icon: Bell },
        ...(isAdmin ? [
            { id: 'smtp', label: 'SMTP / Email', icon: Mail },
            { id: 'stripe', label: 'Stripe Billing', icon: CreditCard },
            { id: 'updates', label: 'Updates', icon: Download },
        ] : []),
    ];

    const scopeOptions = ['read', 'write', 'deploy', 'admin'];

    const fmtDate = (s: string | null) => s ? new Date(s).toLocaleDateString() : '—';
    const fmtTime = (s: string) => new Date(s).toLocaleString();

    return (
        <div className="space-y-6 animate-fade-in">
            <div>
                <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                    <Settings2 className="w-7 h-7 text-nova-400" /> Settings
                </h1>
                <p className="text-surface-200/50 text-sm mt-1">Manage your account and preferences</p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
                {/* Sidebar */}
                <div className="glass-card rounded-2xl p-3 h-fit">
                    {tabs.map(tab => (
                        <button key={tab.id} onClick={() => setActiveTab(tab.id)}
                            className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-medium transition-all mb-1 ${activeTab === tab.id ? 'bg-nova-500/15 text-nova-400' : 'text-surface-200/50 hover:text-white hover:bg-white/5'}`}>
                            <tab.icon className="w-4 h-4" /> {tab.label}
                        </button>
                    ))}
                </div>

                {/* Content */}
                <div className="lg:col-span-3">
                    {loading ? (
                        <div className="glass-card rounded-2xl p-6 flex items-center justify-center py-16">
                            <Loader className="w-6 h-6 text-nova-400 animate-spin" />
                        </div>

                    ) : activeTab === 'profile' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <h3 className="text-lg font-bold text-white">Profile Information</h3>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Full Name</label>
                                    <input value={profile.name} onChange={e => setProfile({ ...profile, name: e.target.value })} className={inputCls} />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Email</label>
                                    <input value={profile.email} onChange={e => setProfile({ ...profile, email: e.target.value })} className={inputCls} />
                                </div>
                            </div>
                            <button onClick={handleSaveProfile} disabled={saving} className={btnPrimary}>
                                {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />} Save Changes
                            </button>
                        </div>

                    ) : activeTab === 'security' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <h3 className="text-lg font-bold text-white">Change Password</h3>
                            <div className="space-y-4 max-w-md">
                                <div><label className="block text-sm font-medium text-surface-200 mb-1.5">Current Password</label>
                                    <input type="password" value={passwords.current} onChange={e => setPasswords({ ...passwords, current: e.target.value })} className={inputCls} placeholder="••••••••" /></div>
                                <div><label className="block text-sm font-medium text-surface-200 mb-1.5">New Password</label>
                                    <input type="password" value={passwords.new_pass} onChange={e => setPasswords({ ...passwords, new_pass: e.target.value })} className={inputCls} placeholder="••••••••" /></div>
                                <div><label className="block text-sm font-medium text-surface-200 mb-1.5">Confirm Password</label>
                                    <input type="password" value={passwords.confirm} onChange={e => setPasswords({ ...passwords, confirm: e.target.value })} className={inputCls} placeholder="••••••••" /></div>
                            </div>
                            <button onClick={handleChangePassword} disabled={saving} className={btnPrimary}>
                                {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Lock className="w-4 h-4" />} Update Password
                            </button>
                        </div>

                    ) : activeTab === '2fa' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <div className="flex items-center justify-between">
                                <h3 className="text-lg font-bold text-white">Two-Factor Authentication</h3>
                                <span className="px-2.5 py-1 rounded-full text-xs bg-surface-700 text-surface-200/60">TOTP</span>
                            </div>
                            <p className="text-sm text-surface-200/60">Add an extra layer of security using an authenticator app (Google Authenticator, Authy, etc.)</p>

                            {!totpSetup ? (
                                <div className="space-y-4">
                                    <button onClick={handleTotpSetup} disabled={totpLoading} className={btnPrimary}>
                                        {totpLoading ? <Loader className="w-4 h-4 animate-spin" /> : <QrCode className="w-4 h-4" />} Set Up 2FA
                                    </button>
                                    <div className="border-t border-surface-700/50 pt-4">
                                        <h4 className="text-sm font-medium text-white mb-3">Disable 2FA</h4>
                                        <div className="flex gap-3 max-w-md">
                                            <input type="password" value={disablePassword} onChange={e => setDisablePassword(e.target.value)}
                                                placeholder="Current password" className={inputCls} />
                                            <button onClick={handleTotpDisable} disabled={totpLoading || !disablePassword}
                                                className="px-4 py-2.5 rounded-xl border border-danger/30 text-danger text-sm hover:bg-danger/10 transition-colors disabled:opacity-40 whitespace-nowrap flex items-center gap-2">
                                                {totpLoading ? <Loader className="w-4 h-4 animate-spin" /> : <X className="w-4 h-4" />} Disable
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ) : (
                                <div className="space-y-5">
                                    <div className="p-4 rounded-xl bg-surface-800/50 border border-surface-700/50">
                                        <p className="text-xs text-surface-200/60 mb-3">Scan this QR code with your authenticator app:</p>
                                        <img src={`https://api.qrserver.com/v1/create-qr-code/?data=${encodeURIComponent(totpSetup.qr_uri)}&size=180x180&bgcolor=0f172a&color=818cf8`}
                                            alt="QR Code" className="rounded-lg mb-3" />
                                        <p className="text-xs text-surface-200/40">Or enter manually: <code className="text-nova-400 font-mono">{totpSetup.secret}</code></p>
                                    </div>
                                    <div className="max-w-xs space-y-3">
                                        <label className="block text-sm font-medium text-surface-200">Enter 6-digit code to confirm</label>
                                        <input value={totpCode} onChange={e => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                                            className={inputCls} placeholder="000000" maxLength={6} autoFocus />
                                        <div className="flex gap-3">
                                            <button onClick={handleTotpEnable} disabled={totpLoading || totpCode.length < 6} className={btnPrimary}>
                                                {totpLoading ? <Loader className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />} Enable 2FA
                                            </button>
                                            <button onClick={() => { setTotpSetup(null); setTotpCode(''); }}
                                                className="px-4 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                                                Cancel
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {backupCodes.length > 0 && (
                                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm">
                                    <div className="glass-card rounded-2xl p-6 w-full max-w-md mx-4 space-y-4">
                                        <h3 className="text-lg font-bold text-white flex items-center gap-2"><Shield className="w-5 h-5 text-nova-400" /> Backup Codes</h3>
                                        <p className="text-sm text-surface-200/60">Save these codes in a safe place. Each can be used once if you lose access to your authenticator.</p>
                                        <div className="grid grid-cols-2 gap-2">
                                            {backupCodes.map((c, i) => (
                                                <code key={i} className="px-3 py-1.5 bg-surface-800 rounded-lg text-sm font-mono text-nova-300 text-center">{c}</code>
                                            ))}
                                        </div>
                                        <button onClick={() => setBackupCodes([])} className={btnPrimary + ' w-full justify-center'}>
                                            <Check className="w-4 h-4" /> I've saved my backup codes
                                        </button>
                                    </div>
                                </div>
                            )}
                        </div>

                    ) : activeTab === 'api-keys' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-5">
                            <div className="flex items-center justify-between">
                                <h3 className="text-lg font-bold text-white">API Keys</h3>
                                <button onClick={() => setShowNewKey(true)} className={btnPrimary}>
                                    <Plus className="w-4 h-4" /> New Key
                                </button>
                            </div>
                            <p className="text-sm text-surface-200/60">Use API keys to authenticate programmatic access. Keys are shown only once at creation.</p>

                            {keysLoading ? (
                                <div className="flex items-center justify-center py-8"><Loader className="w-5 h-5 text-nova-400 animate-spin" /></div>
                            ) : apiKeys.length === 0 ? (
                                <div className="text-center py-8 text-surface-200/40 text-sm">No API keys yet</div>
                            ) : (
                                <div className="space-y-2">
                                    {apiKeys.map(k => (
                                        <div key={k.id} className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
                                            <div>
                                                <p className="text-sm font-medium text-white">{k.name}</p>
                                                <p className="text-xs text-surface-200/50 font-mono mt-0.5">{k.key_prefix}...</p>
                                                <div className="flex items-center gap-3 mt-1">
                                                    <span className="text-[10px] text-surface-200/40">Created {fmtDate(k.created_at)}</span>
                                                    {k.last_used_at && <span className="text-[10px] text-surface-200/40">Last used {fmtDate(k.last_used_at)}</span>}
                                                    {k.expires_at && <span className="text-[10px] text-warning">Expires {fmtDate(k.expires_at)}</span>}
                                                </div>
                                                <div className="flex gap-1 mt-1.5">
                                                    {k.scopes?.map((s: string) => (
                                                        <span key={s} className="px-1.5 py-0.5 rounded text-[10px] bg-nova-500/10 text-nova-400">{s}</span>
                                                    ))}
                                                </div>
                                            </div>
                                            <button onClick={() => handleRevokeKey(k.id)}
                                                className="p-2 rounded-lg text-danger hover:bg-danger/10 transition-colors">
                                                <Trash2 className="w-4 h-4" />
                                            </button>
                                        </div>
                                    ))}
                                </div>
                            )}

                            {showNewKey && (
                                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowNewKey(false)}>
                                    <div className="glass-card rounded-2xl p-6 w-full max-w-md mx-4 space-y-4" onClick={e => e.stopPropagation()}>
                                        <div className="flex items-center justify-between">
                                            <h3 className="text-lg font-bold text-white">Create API Key</h3>
                                            <button onClick={() => setShowNewKey(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">Key Name</label>
                                            <input value={newKeyName} onChange={e => setNewKeyName(e.target.value)} className={inputCls} placeholder="e.g. CI/CD Pipeline" autoFocus />
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-2">Scopes</label>
                                            <div className="flex flex-wrap gap-2">
                                                {scopeOptions.map(s => (
                                                    <button key={s} type="button"
                                                        onClick={() => setNewKeyScopes(prev => prev.includes(s) ? prev.filter(x => x !== s) : [...prev, s])}
                                                        className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${newKeyScopes.includes(s) ? 'bg-nova-500/20 text-nova-400 border border-nova-500/30' : 'bg-surface-800 text-surface-200/60 border border-surface-700/50 hover:border-surface-600'}`}>
                                                        {s}
                                                    </button>
                                                ))}
                                            </div>
                                        </div>
                                        <div className="flex gap-3 pt-2">
                                            <button onClick={() => setShowNewKey(false)} className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">Cancel</button>
                                            <button onClick={handleCreateKey} disabled={saving || !newKeyName.trim()} className={btnPrimary + ' flex-1 justify-center'}>
                                                {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Key className="w-4 h-4" />} Create
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            )}

                            {createdKey && (
                                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm">
                                    <div className="glass-card rounded-2xl p-6 w-full max-w-md mx-4 space-y-4">
                                        <h3 className="text-lg font-bold text-white flex items-center gap-2"><Key className="w-5 h-5 text-nova-400" /> API Key Created</h3>
                                        <p className="text-sm text-warning">Copy this key now — it won't be shown again.</p>
                                        <div className="flex items-center gap-2 p-3 bg-surface-800/80 rounded-xl border border-surface-700/50">
                                            <code className="flex-1 text-xs font-mono text-nova-300 break-all">{createdKey}</code>
                                            <button onClick={handleCopyKey} className="p-1.5 rounded-lg hover:bg-surface-700 text-surface-200/60 hover:text-white transition-colors flex-shrink-0">
                                                {keyCopied ? <Check className="w-4 h-4 text-success" /> : <Copy className="w-4 h-4" />}
                                            </button>
                                        </div>
                                        <button onClick={() => setCreatedKey(null)} className={btnPrimary + ' w-full justify-center'}>
                                            <Check className="w-4 h-4" /> Done
                                        </button>
                                    </div>
                                </div>
                            )}
                        </div>

                    ) : activeTab === 'sessions' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-5">
                            <div className="flex items-center justify-between">
                                <h3 className="text-lg font-bold text-white">Active Sessions</h3>
                                <button onClick={handleRevokeAll}
                                    className="px-4 py-2 rounded-xl border border-danger/30 text-danger text-sm hover:bg-danger/10 transition-colors flex items-center gap-2">
                                    <LogOut className="w-4 h-4" /> Revoke All Others
                                </button>
                            </div>

                            {sessionsLoading ? (
                                <div className="flex items-center justify-center py-8"><Loader className="w-5 h-5 text-nova-400 animate-spin" /></div>
                            ) : sessions.length === 0 ? (
                                <div className="text-center py-8 text-surface-200/40 text-sm">No active sessions</div>
                            ) : (
                                <div className="space-y-2">
                                    {sessions.map(s => (
                                        <div key={s.id} className="flex items-start justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
                                            <div>
                                                <p className="text-sm font-medium text-white">{s.ip_address || 'Unknown IP'}</p>
                                                <p className="text-xs text-surface-200/50 mt-0.5 max-w-xs truncate">{s.user_agent || 'Unknown browser'}</p>
                                                <p className="text-[10px] text-surface-200/40 mt-1">
                                                    Started {fmtTime(s.created_at)} · Last seen {fmtTime(s.last_seen_at)}
                                                </p>
                                            </div>
                                            <button onClick={() => handleRevokeSession(s.id)}
                                                className="p-2 rounded-lg text-danger hover:bg-danger/10 transition-colors flex-shrink-0 ml-3">
                                                <X className="w-4 h-4" />
                                            </button>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>

                    ) : activeTab === 'smtp' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <div>
                                <h3 className="text-lg font-bold text-white flex items-center gap-2"><Mail className="w-5 h-5 text-nova-400" /> SMTP Configuration</h3>
                                <p className="text-sm text-surface-200/50 mt-1">Used for password reset and alert notification emails. Changes take effect immediately.</p>
                            </div>
                            {systemLoading ? (
                                <div className="flex items-center justify-center py-10"><Loader className="w-5 h-5 text-nova-400 animate-spin" /></div>
                            ) : (
                                <div className="space-y-4 max-w-lg">
                                    <div className="grid grid-cols-3 gap-3">
                                        <div className="col-span-2">
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">SMTP Host</label>
                                            <input value={smtpSettings.smtp_host} onChange={e => setSmtpSettings(s => ({ ...s, smtp_host: e.target.value }))}
                                                className={inputCls} placeholder="smtp.gmail.com" />
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">Port</label>
                                            <input value={smtpSettings.smtp_port} onChange={e => setSmtpSettings(s => ({ ...s, smtp_port: e.target.value }))}
                                                className={inputCls} placeholder="587" />
                                        </div>
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Username</label>
                                        <input value={smtpSettings.smtp_user} onChange={e => setSmtpSettings(s => ({ ...s, smtp_user: e.target.value }))}
                                            className={inputCls} placeholder="you@example.com" />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Password</label>
                                        <input type="password" value={smtpSettings.smtp_password} onChange={e => setSmtpSettings(s => ({ ...s, smtp_password: e.target.value }))}
                                            className={inputCls} placeholder={smtpSettings.smtp_password === '••••••••' ? 'Saved — enter new password to change' : '••••••••'} />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">From Address</label>
                                        <input value={smtpSettings.smtp_from} onChange={e => setSmtpSettings(s => ({ ...s, smtp_from: e.target.value }))}
                                            className={inputCls} placeholder="NovaPanel <noreply@example.com>" />
                                    </div>
                                    <div className="flex gap-3 pt-2">
                                        <button onClick={handleSaveSmtp} disabled={saving} className={btnPrimary}>
                                            {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />} Save
                                        </button>
                                        <button onClick={handleTestSmtp} disabled={testingSmtp || saving} className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-white/10 text-surface-200/70 text-sm hover:bg-white/5 transition-colors disabled:opacity-50">
                                            {testingSmtp ? <Loader className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />} Send Test Email
                                        </button>
                                    </div>
                                </div>
                            )}
                        </div>

                    ) : activeTab === 'stripe' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <div>
                                <h3 className="text-lg font-bold text-white flex items-center gap-2"><CreditCard className="w-5 h-5 text-nova-400" /> Stripe Configuration</h3>
                                <p className="text-sm text-surface-200/50 mt-1">Required for subscription billing. Get your keys from the <span className="text-nova-400">Stripe Dashboard</span>.</p>
                            </div>
                            {systemLoading ? (
                                <div className="flex items-center justify-center py-10"><Loader className="w-5 h-5 text-nova-400 animate-spin" /></div>
                            ) : (
                                <div className="space-y-4 max-w-lg">
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Secret Key</label>
                                        <input type="password" value={stripeSettings.stripe_secret_key} onChange={e => setStripeSettings(s => ({ ...s, stripe_secret_key: e.target.value }))}
                                            className={inputCls} placeholder={stripeSettings.stripe_secret_key === '••••••••' ? 'Saved — enter new key to change' : 'sk_live_...'} />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Webhook Secret</label>
                                        <input type="password" value={stripeSettings.stripe_webhook_secret} onChange={e => setStripeSettings(s => ({ ...s, stripe_webhook_secret: e.target.value }))}
                                            className={inputCls} placeholder={stripeSettings.stripe_webhook_secret === '••••••••' ? 'Saved — enter new secret to change' : 'whsec_...'} />
                                    </div>
                                    <div className="grid grid-cols-2 gap-3">
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">Enterprise Price ID</label>
                                            <input value={stripeSettings.stripe_price_enterprise} onChange={e => setStripeSettings(s => ({ ...s, stripe_price_enterprise: e.target.value }))}
                                                className={inputCls} placeholder="price_..." />
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">Reseller Price ID</label>
                                            <input value={stripeSettings.stripe_price_reseller} onChange={e => setStripeSettings(s => ({ ...s, stripe_price_reseller: e.target.value }))}
                                                className={inputCls} placeholder="price_..." />
                                        </div>
                                    </div>
                                    <div className="p-3 rounded-xl bg-surface-800/50 border border-surface-700/30 text-xs text-surface-200/50 space-y-1">
                                        <p>Webhook endpoint to register in Stripe Dashboard:</p>
                                        <code className="text-nova-400 font-mono">{window.location.origin}/api/v1/billing/webhook</code>
                                    </div>
                                    <button onClick={handleSaveStripe} disabled={saving} className={btnPrimary}>
                                        {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />} Save Stripe Settings
                                    </button>
                                </div>
                            )}
                        </div>

                    ) : activeTab === 'updates' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h3 className="text-lg font-bold text-white flex items-center gap-2">
                                        <Zap className="w-5 h-5 text-nova-400" /> Software Updates
                                    </h3>
                                    <p className="text-sm text-surface-200/50 mt-1">Keep NovaPanel up to date with the latest features and security fixes.</p>
                                </div>
                                <button onClick={handleCheckUpdates} disabled={checking} className="flex items-center gap-2 px-4 py-2 rounded-xl bg-surface-700/50 text-sm text-surface-200/70 hover:text-white hover:bg-surface-700 transition-all disabled:opacity-50">
                                    <RefreshCw className={`w-4 h-4 ${checking ? 'animate-spin' : ''}`} /> Check Now
                                </button>
                            </div>

                            {updateLoading ? (
                                <div className="flex items-center justify-center py-10"><Loader className="w-5 h-5 text-nova-400 animate-spin" /></div>
                            ) : updateStatus ? (
                                <>
                                    {/* Version cards */}
                                    <div className="grid grid-cols-2 gap-4">
                                        <div className="p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
                                            <p className="text-xs text-surface-200/40 uppercase tracking-wider mb-1">Current Version</p>
                                            <p className="text-lg font-bold text-white font-mono">{updateStatus.current_version}</p>
                                        </div>
                                        <div className={`p-4 rounded-xl border ${updateStatus.update_available ? 'bg-nova-500/10 border-nova-500/30' : 'bg-surface-800/50 border-surface-700/30'}`}>
                                            <p className="text-xs text-surface-200/40 uppercase tracking-wider mb-1">Latest Version</p>
                                            <p className={`text-lg font-bold font-mono ${updateStatus.update_available ? 'text-nova-400' : 'text-white'}`}>
                                                {updateStatus.latest_version || '—'}
                                            </p>
                                        </div>
                                    </div>

                                    {/* Status banner */}
                                    {updateStatus.update_available ? (
                                        <div className="flex items-start gap-3 p-4 rounded-xl bg-nova-500/10 border border-nova-500/30">
                                            <AlertTriangle className="w-5 h-5 text-nova-400 flex-shrink-0 mt-0.5" />
                                            <div className="flex-1 min-w-0">
                                                <p className="text-sm font-medium text-nova-300">Update available: {updateStatus.latest_version}</p>
                                                <p className="text-xs text-surface-200/50 mt-0.5">
                                                    {updateStatus.last_checked ? `Last checked ${new Date(updateStatus.last_checked).toLocaleString()}` : ''}
                                                </p>
                                            </div>
                                            {updateStatus.release_url && (
                                                <a href={updateStatus.release_url} target="_blank" rel="noopener noreferrer"
                                                    className="flex items-center gap-1 text-xs text-nova-400 hover:text-nova-300 transition-colors flex-shrink-0">
                                                    <ExternalLink className="w-3 h-3" /> Release
                                                </a>
                                            )}
                                        </div>
                                    ) : (
                                        <div className="flex items-center gap-3 p-4 rounded-xl bg-green-500/10 border border-green-500/20">
                                            <CheckCircle2 className="w-5 h-5 text-green-400 flex-shrink-0" />
                                            <p className="text-sm text-green-300">NovaPanel is up to date</p>
                                        </div>
                                    )}

                                    {/* Release notes */}
                                    {updateStatus.release_notes && (
                                        <div>
                                            <p className="text-sm font-medium text-surface-200 mb-2">Release Notes</p>
                                            <div className="p-4 rounded-xl bg-surface-800/50 border border-surface-700/30 text-sm text-surface-200/70 whitespace-pre-wrap font-mono leading-relaxed max-h-48 overflow-y-auto">
                                                {updateStatus.release_notes}
                                            </div>
                                        </div>
                                    )}

                                    {/* Apply log */}
                                    {applyLog.length > 0 && (
                                        <div>
                                            <p className="text-sm font-medium text-surface-200 mb-2">Update Progress</p>
                                            <div className="p-4 rounded-xl bg-black/40 border border-surface-700/30 font-mono text-xs text-green-400 space-y-1 max-h-40 overflow-y-auto">
                                                {applyLog.map((line, i) => <div key={i}>{line}</div>)}
                                            </div>
                                        </div>
                                    )}

                                    {/* Apply button */}
                                    {updateStatus.update_available && applyLog.length === 0 && (
                                        <button onClick={handleApplyUpdate} disabled={updateStatus.applying}
                                            className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-50">
                                            {updateStatus.applying
                                                ? <><Loader className="w-4 h-4 animate-spin" /> Applying...</>
                                                : <><Download className="w-4 h-4" /> Apply Update to {updateStatus.latest_version}</>
                                            }
                                        </button>
                                    )}
                                </>
                            ) : (
                                <p className="text-sm text-surface-200/50">Click "Check Now" to look for updates.</p>
                            )}
                        </div>

                    ) : (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <h3 className="text-lg font-bold text-white">Notification Preferences</h3>
                            <div className="space-y-4">
                                {['Deployment alerts', 'Security warnings', 'Backup notifications', 'Billing reminders', 'Server health alerts'].map((label, i) => (
                                    <div key={i} className="flex items-center justify-between py-2">
                                        <span className="text-sm text-surface-200/70">{label}</span>
                                        <button className={`w-11 h-6 rounded-full transition-colors relative ${i < 3 ? 'bg-nova-500' : 'bg-surface-700'}`}>
                                            <div className={`w-4 h-4 rounded-full bg-white absolute top-1 transition-all ${i < 3 ? 'left-6' : 'left-1'}`} />
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
