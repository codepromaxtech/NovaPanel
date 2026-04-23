import { useState, useEffect } from 'react';
import { Settings2, User, Lock, Bell, Save, Loader, Key, Shield, Monitor, Plus, Trash2, Copy, Check, QrCode, X, LogOut } from 'lucide-react';
import { settingsService } from '../services/billing';
import { authService } from '../services/auth';
import { useToast } from '../components/ui/ToastProvider';
import { useModalLock } from '../hooks/useModalLock';

const inputCls = 'w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20';
const btnPrimary = 'flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-50';

export default function Settings() {
    const toast = useToast();
    const [activeTab, setActiveTab] = useState('profile');
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

    useEffect(() => {
        settingsService.getProfile()
            .then(data => setProfile({ name: data.name || '', email: data.email || '' }))
            .catch(() => setProfile({ name: 'Admin User', email: 'admin@novapanel.io' }))
            .finally(() => setLoading(false));
    }, []);

    useEffect(() => {
        if (activeTab === 'api-keys') loadApiKeys();
        if (activeTab === 'sessions') loadSessions();
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

    const tabs = [
        { id: 'profile', label: 'Profile', icon: User },
        { id: 'security', label: 'Password', icon: Lock },
        { id: '2fa', label: 'Two-Factor Auth', icon: Shield },
        { id: 'api-keys', label: 'API Keys', icon: Key },
        { id: 'sessions', label: 'Sessions', icon: Monitor },
        { id: 'notifications', label: 'Notifications', icon: Bell },
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
