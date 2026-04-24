import { useState, useEffect } from 'react';
import {
    HardDrive, Plus, Trash2, X, Eye, EyeOff, Server, User, Folder,
    Database, Key, ChevronDown, ChevronRight, Copy, Check,
} from 'lucide-react';
import { ftpService, FTPAccount, SFTPKey } from '../services/ftp';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface ServerEntry { id: string; name: string; ip_address: string; }

const inputCls = 'w-full px-3.5 py-2.5 rounded-xl bg-surface-700/50 border border-surface-600/50 text-white text-sm placeholder:text-surface-400/50 focus:outline-none focus:border-nova-500/70 focus:ring-1 focus:ring-nova-500/30 transition-all';

function useCopy() {
    const [copied, setCopied] = useState('');
    const copy = (text: string, id: string) => {
        navigator.clipboard.writeText(text);
        setCopied(id);
        setTimeout(() => setCopied(''), 2000);
    };
    return { copied, copy };
}

// ─── SFTP Keys Panel ─────────────────────────────────────────────────────────
function SFTPKeysPanel({ acc, serverId }: { acc: FTPAccount; serverId: string }) {
    const toast = useToast();
    const { copied, copy } = useCopy();
    const [keys, setKeys] = useState<SFTPKey[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAdd, setShowAdd] = useState(false);
    const [label, setLabel] = useState('');
    const [pubKey, setPubKey] = useState('');
    const [adding, setAdding] = useState(false);

    const load = async () => {
        setLoading(true);
        try { setKeys(await ftpService.listKeys(serverId, acc.id)); }
        catch { setKeys([]); }
        finally { setLoading(false); }
    };

    useEffect(() => { load(); }, [acc.id]);

    const handleAdd = async () => {
        if (!pubKey.trim()) return;
        setAdding(true);
        try {
            await ftpService.addKey(serverId, acc.id, { label: label || 'key', public_key: pubKey.trim() });
            toast.success('SSH key added — provisioning on server');
            setShowAdd(false); setLabel(''); setPubKey('');
            load();
        } catch (err: any) {
            toast.error(err.response?.data?.error || 'Invalid public key');
        } finally { setAdding(false); }
    };

    const handleDelete = async (keyID: string) => {
        if (!confirm('Remove this SSH key?')) return;
        try { await ftpService.deleteKey(serverId, acc.id, keyID); toast.success('Key removed'); load(); }
        catch { toast.error('Failed to remove key'); }
    };

    return (
        <div className="mt-3 bg-surface-900/50 rounded-xl p-4 space-y-3">
            <div className="flex items-center justify-between">
                <p className="text-xs font-medium text-surface-300 flex items-center gap-1.5">
                    <Key className="w-3.5 h-3.5 text-nova-400" /> SSH Public Keys for SFTP
                </p>
                <button onClick={() => setShowAdd(s => !s)}
                    className="text-xs text-nova-400 hover:text-nova-300 transition-colors flex items-center gap-1">
                    <Plus className="w-3 h-3" /> Add Key
                </button>
            </div>

            {showAdd && (
                <div className="space-y-2 p-3 rounded-lg bg-surface-800/60 border border-surface-700/40">
                    <input value={label} onChange={e => setLabel(e.target.value)}
                        placeholder="Key label (e.g. my-laptop)" className={inputCls + ' text-xs py-2'} />
                    <textarea value={pubKey} onChange={e => setPubKey(e.target.value)}
                        placeholder="ssh-rsa AAAA... or ssh-ed25519 AAAA..."
                        className={inputCls + ' text-xs py-2 font-mono h-20 resize-none'} />
                    <div className="flex gap-2">
                        <button onClick={handleAdd} disabled={adding || !pubKey.trim()}
                            className="flex-1 py-1.5 rounded-lg bg-nova-600 text-white text-xs font-medium hover:bg-nova-500 disabled:opacity-50 transition-colors flex items-center justify-center gap-1">
                            {adding ? <div className="w-3 h-3 border border-white/30 border-t-white rounded-full animate-spin" /> : <Plus className="w-3 h-3" />}
                            Add
                        </button>
                        <button onClick={() => { setShowAdd(false); setLabel(''); setPubKey(''); }}
                            className="px-3 py-1.5 rounded-lg text-surface-400 hover:text-white text-xs transition-colors">
                            Cancel
                        </button>
                    </div>
                </div>
            )}

            {loading ? (
                <p className="text-xs text-surface-500">Loading keys…</p>
            ) : keys.length === 0 ? (
                <p className="text-xs text-surface-500 italic">No SSH keys — add one to enable key-based SFTP login</p>
            ) : (
                <div className="space-y-1.5">
                    {keys.map(k => (
                        <div key={k.id} className="flex items-center justify-between p-2.5 rounded-lg bg-surface-800/40 border border-surface-700/30 group">
                            <div>
                                <p className="text-xs font-medium text-surface-200">{k.label}</p>
                                <p className="text-[10px] font-mono text-surface-400 mt-0.5">{k.fingerprint}</p>
                            </div>
                            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                <button onClick={() => copy(k.fingerprint, k.id)}
                                    className="p-1 rounded text-surface-400 hover:text-nova-400 transition-colors" title="Copy fingerprint">
                                    {copied === k.id ? <Check className="w-3.5 h-3.5 text-success" /> : <Copy className="w-3.5 h-3.5" />}
                                </button>
                                <button onClick={() => handleDelete(k.id)}
                                    className="p-1 rounded text-surface-400 hover:text-danger transition-colors" title="Remove key">
                                    <Trash2 className="w-3.5 h-3.5" />
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}

// ─── Main FTP Page ────────────────────────────────────────────────────────────
export default function FTP() {
    const toast = useToast();
    const [accounts, setAccounts] = useState<FTPAccount[]>([]);
    const [servers, setServers] = useState<ServerEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [creating, setCreating] = useState(false);
    const [showPass, setShowPass] = useState(false);
    const [filterServer, setFilterServer] = useState('');
    const [expandedId, setExpandedId] = useState<string | null>(null);
    const [form, setForm] = useState({ server_id: '', username: '', password: '', home_dir: '', quota_mb: 1024 });
    const [formErr, setFormErr] = useState('');

    const load = async () => {
        try {
            const [accs, srvs] = await Promise.all([ftpService.list(), serverService.list()]);
            setAccounts(Array.isArray(accs) ? accs : []);
            const srvList = Array.isArray(srvs) ? srvs : (srvs?.data || []);
            setServers(srvList);
            if (srvList.length > 0 && !form.server_id) setForm(f => ({ ...f, server_id: srvList[0].id }));
        } catch { setAccounts([]); }
        finally { setLoading(false); }
    };

    useEffect(() => { load(); }, []);

    const serverName = (id: string) => servers.find(s => s.id === id)?.name || id.slice(0, 8);
    const displayed = filterServer ? accounts.filter(a => a.server_id === filterServer) : accounts;

    const handleCreate = async () => {
        if (!form.server_id) { setFormErr('Select a server.'); return; }
        if (!form.username) { setFormErr('Username is required.'); return; }
        if (form.password.length < 8) { setFormErr('Password must be at least 8 characters.'); return; }
        setCreating(true); setFormErr('');
        try {
            await ftpService.create(form.server_id, {
                username: form.username, password: form.password,
                home_dir: form.home_dir || undefined, quota_mb: form.quota_mb || 1024,
            });
            toast.success('FTP/SFTP account created — provisioning in background');
            setShowCreate(false);
            setForm(f => ({ ...f, username: '', password: '', home_dir: '', quota_mb: 1024 }));
            load();
        } catch (err: any) {
            setFormErr(err.response?.data?.error || 'Failed to create account');
        } finally { setCreating(false); }
    };

    const handleDelete = async (acc: FTPAccount) => {
        if (!confirm(`Delete ftp_${acc.username}? This removes the system user from the server.`)) return;
        try { await ftpService.delete(acc.server_id, acc.id); toast.success('Account deleted'); load(); }
        catch { toast.error('Failed to delete account'); }
    };

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <HardDrive className="w-7 h-7 text-nova-400" /> FTP / SFTP Accounts
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Manage FTP and SFTP access across your servers</p>
                </div>
                <button onClick={() => setShowCreate(true)}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                    <Plus className="w-4 h-4" /> New Account
                </button>
            </div>

            {/* Filter */}
            {servers.length > 1 && (
                <div className="flex items-center gap-3">
                    <Server className="w-4 h-4 text-surface-400" />
                    <select value={filterServer} onChange={e => setFilterServer(e.target.value)}
                        className="px-3 py-1.5 rounded-lg bg-surface-700/50 border border-surface-600/50 text-white text-sm focus:outline-none focus:border-nova-500/50">
                        <option value="">All Servers</option>
                        {servers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                    </select>
                    <span className="text-surface-400 text-xs">{displayed.length} account{displayed.length !== 1 ? 's' : ''}</span>
                </div>
            )}

            {/* Accounts */}
            <div className="rounded-2xl border border-surface-700/50 bg-surface-800/40 backdrop-blur-sm overflow-hidden">
                {loading ? (
                    <div className="flex items-center justify-center py-16 text-surface-400">
                        <div className="w-6 h-6 border-2 border-surface-500 border-t-nova-400 rounded-full animate-spin mr-3" />
                        Loading…
                    </div>
                ) : displayed.length === 0 ? (
                    <div className="text-center py-16 text-surface-400">
                        <HardDrive className="w-12 h-12 mx-auto mb-4 opacity-30" />
                        <p className="font-medium">No FTP/SFTP accounts</p>
                        <p className="text-sm mt-1 opacity-60">Create an account to get started</p>
                    </div>
                ) : (
                    <div className="divide-y divide-surface-700/30">
                        {displayed.map(acc => (
                            <div key={acc.id}>
                                {/* Row */}
                                <div className="flex items-center gap-4 px-5 py-3.5 hover:bg-surface-700/20 transition-colors">
                                    <button onClick={() => setExpandedId(expandedId === acc.id ? null : acc.id)}
                                        className="text-surface-400 hover:text-nova-400 transition-colors flex-shrink-0">
                                        {expandedId === acc.id ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                                    </button>
                                    <div className="flex items-center gap-2 w-44 min-w-0">
                                        <User className="w-4 h-4 text-nova-400 flex-shrink-0" />
                                        <span className="font-mono text-white text-sm truncate">ftp_{acc.username}</span>
                                    </div>
                                    <div className="flex items-center gap-1.5 text-surface-300 text-sm w-32">
                                        <Server className="w-3.5 h-3.5 text-surface-400 flex-shrink-0" />
                                        <span className="truncate">{serverName(acc.server_id)}</span>
                                    </div>
                                    <div className="flex items-center gap-1.5 text-surface-400 font-mono text-xs flex-1 min-w-0">
                                        <Folder className="w-3.5 h-3.5 flex-shrink-0" />
                                        <span className="truncate">{acc.home_dir}</span>
                                    </div>
                                    <div className="flex items-center gap-1.5 text-surface-400 text-xs w-20">
                                        <Database className="w-3.5 h-3.5" />
                                        {acc.quota_mb >= 1024 ? `${(acc.quota_mb / 1024).toFixed(0)} GB` : `${acc.quota_mb} MB`}
                                    </div>
                                    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium flex-shrink-0 ${acc.is_active ? 'bg-success/10 text-success' : 'bg-surface-600/50 text-surface-400'}`}>
                                        <span className={`w-1.5 h-1.5 rounded-full ${acc.is_active ? 'bg-success' : 'bg-surface-400'}`} />
                                        {acc.is_active ? 'Active' : 'Inactive'}
                                    </span>
                                    <button onClick={() => handleDelete(acc)}
                                        className="p-1.5 rounded-lg text-surface-400 hover:text-danger hover:bg-danger/10 transition-colors flex-shrink-0">
                                        <Trash2 className="w-4 h-4" />
                                    </button>
                                </div>

                                {/* SFTP Keys panel */}
                                {expandedId === acc.id && (
                                    <div className="px-5 pb-4">
                                        <SFTPKeysPanel acc={acc} serverId={acc.server_id} />
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Info */}
            <div className="rounded-xl border border-surface-700/30 bg-surface-800/20 p-4 text-sm text-surface-400">
                <p className="font-medium text-surface-300 mb-1">How it works</p>
                <ul className="space-y-1 text-xs">
                    <li>• A system user <span className="font-mono text-surface-300">ftp_&lt;username&gt;</span> is created on the server via SSH</li>
                    <li>• <strong className="text-surface-300">FTP</strong>: password auth via vsftpd (port 21) — chroot-jailed to home directory</li>
                    <li>• <strong className="text-surface-300">SFTP</strong>: SSH key auth via the native SSH subsystem (port 22) — expand a row to manage keys</li>
                    <li>• Click any row to expand and manage SSH public keys for that account</li>
                </ul>
            </div>

            {/* Create Modal */}
            {showCreate && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
                    <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setShowCreate(false)} />
                    <div className="relative z-10 w-full max-w-md rounded-2xl bg-surface-800 border border-surface-700/50 shadow-2xl p-6">
                        <div className="flex items-center justify-between mb-5">
                            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                                <HardDrive className="w-5 h-5 text-nova-400" /> New FTP / SFTP Account
                            </h2>
                            <button onClick={() => setShowCreate(false)} className="text-surface-400 hover:text-white transition-colors"><X className="w-5 h-5" /></button>
                        </div>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Server</label>
                                <select value={form.server_id} onChange={e => setForm(f => ({ ...f, server_id: e.target.value }))}
                                    className={inputCls + ' bg-surface-700/50'}>
                                    <option value="">Select server…</option>
                                    {servers.map(s => <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>)}
                                </select>
                            </div>
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Username</label>
                                <div className="flex">
                                    <span className="px-3 py-2.5 rounded-l-xl bg-surface-700 border border-r-0 border-surface-600/50 text-surface-400 text-sm font-mono">ftp_</span>
                                    <input value={form.username}
                                        onChange={e => setForm(f => ({ ...f, username: e.target.value.replace(/[^a-z0-9_-]/gi, '') }))}
                                        placeholder="johndoe" className={inputCls + ' rounded-l-none'} />
                                </div>
                            </div>
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Password <span className="text-surface-500">(for FTP auth)</span></label>
                                <div className="relative">
                                    <input type={showPass ? 'text' : 'password'} value={form.password}
                                        onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                                        placeholder="Min 8 characters" className={inputCls + ' pr-10'} />
                                    <button type="button" onClick={() => setShowPass(p => !p)}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 text-surface-400 hover:text-white">
                                        {showPass ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                    </button>
                                </div>
                            </div>
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Home Directory <span className="text-surface-500">(optional)</span></label>
                                <input value={form.home_dir}
                                    onChange={e => setForm(f => ({ ...f, home_dir: e.target.value }))}
                                    placeholder={`/var/www/${form.username || 'username'}`} className={inputCls} />
                            </div>
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">
                                    Disk Quota — <span className="text-nova-400">{form.quota_mb >= 1024 ? `${(form.quota_mb / 1024).toFixed(0)} GB` : `${form.quota_mb} MB`}</span>
                                </label>
                                <input type="range" min={256} max={51200} step={256} value={form.quota_mb}
                                    onChange={e => setForm(f => ({ ...f, quota_mb: Number(e.target.value) }))}
                                    className="w-full accent-nova-500" />
                                <div className="flex justify-between text-xs text-surface-500 mt-1"><span>256 MB</span><span>50 GB</span></div>
                            </div>
                            {formErr && <p className="text-danger text-xs flex items-center gap-1.5"><span className="w-1 h-1 rounded-full bg-danger" /> {formErr}</p>}
                            <button onClick={handleCreate} disabled={creating}
                                className="w-full py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-60 flex items-center justify-center gap-2">
                                {creating ? <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Creating…</> : <><Plus className="w-4 h-4" /> Create Account</>}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
