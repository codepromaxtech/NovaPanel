import { useState, useEffect } from 'react';
import { HardDrive, Plus, Trash2, X, Eye, EyeOff, Server, User, Folder, Database } from 'lucide-react';
import { ftpService, FTPAccount } from '../services/ftp';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

interface Server { id: string; name: string; ip_address: string; }

const inputCls = 'w-full px-3.5 py-2.5 rounded-xl bg-surface-700/50 border border-surface-600/50 text-white text-sm placeholder:text-surface-400/50 focus:outline-none focus:border-nova-500/70 focus:ring-1 focus:ring-nova-500/30 transition-all';

export default function FTP() {
    const toast = useToast();
    const [accounts, setAccounts] = useState<FTPAccount[]>([]);
    const [servers, setServers] = useState<Server[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [creating, setCreating] = useState(false);
    const [showPass, setShowPass] = useState(false);
    const [filterServer, setFilterServer] = useState('');
    const [form, setForm] = useState({
        server_id: '', username: '', password: '', home_dir: '', quota_mb: 1024,
    });
    const [formErr, setFormErr] = useState('');

    const load = async () => {
        try {
            const [accs, srvs] = await Promise.all([
                ftpService.list(),
                serverService.list(),
            ]);
            setAccounts(Array.isArray(accs) ? accs : []);
            const srvList = Array.isArray(srvs) ? srvs : (srvs?.data || []);
            setServers(srvList);
            if (srvList.length > 0 && !form.server_id) {
                setForm(f => ({ ...f, server_id: srvList[0].id }));
            }
        } catch {
            setAccounts([]);
        } finally {
            setLoading(false);
        }
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
                username: form.username,
                password: form.password,
                home_dir: form.home_dir || undefined,
                quota_mb: form.quota_mb || 1024,
            });
            toast.success('FTP account created — provisioning on server in background');
            setShowCreate(false);
            setForm(f => ({ ...f, username: '', password: '', home_dir: '', quota_mb: 1024 }));
            load();
        } catch (err: any) {
            setFormErr(err.response?.data?.error || 'Failed to create FTP account');
        } finally {
            setCreating(false);
        }
    };

    const handleDelete = async (acc: FTPAccount) => {
        if (!confirm(`Delete FTP account ftp_${acc.username}? This will remove the system user from the server.`)) return;
        try {
            await ftpService.delete(acc.server_id, acc.id);
            toast.success('FTP account deleted');
            load();
        } catch {
            toast.error('Failed to delete FTP account');
        }
    };

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <HardDrive className="w-7 h-7 text-nova-400" />
                        FTP / SFTP Accounts
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Manage FTP access accounts across your servers</p>
                </div>
                <button
                    onClick={() => setShowCreate(true)}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all"
                >
                    <Plus className="w-4 h-4" /> New Account
                </button>
            </div>

            {/* Filter bar */}
            {servers.length > 1 && (
                <div className="flex items-center gap-3">
                    <Server className="w-4 h-4 text-surface-400" />
                    <select
                        value={filterServer}
                        onChange={e => setFilterServer(e.target.value)}
                        className="px-3 py-1.5 rounded-lg bg-surface-700/50 border border-surface-600/50 text-white text-sm focus:outline-none focus:border-nova-500/50"
                    >
                        <option value="">All Servers</option>
                        {servers.map(s => (
                            <option key={s.id} value={s.id}>{s.name}</option>
                        ))}
                    </select>
                    <span className="text-surface-400 text-xs">{displayed.length} account{displayed.length !== 1 ? 's' : ''}</span>
                </div>
            )}

            {/* Accounts table */}
            <div className="rounded-2xl border border-surface-700/50 bg-surface-800/40 backdrop-blur-sm overflow-hidden">
                {loading ? (
                    <div className="flex items-center justify-center py-16 text-surface-400">
                        <div className="w-6 h-6 border-2 border-surface-500 border-t-nova-400 rounded-full animate-spin mr-3" />
                        Loading FTP accounts…
                    </div>
                ) : displayed.length === 0 ? (
                    <div className="text-center py-16 text-surface-400">
                        <HardDrive className="w-12 h-12 mx-auto mb-4 opacity-30" />
                        <p className="font-medium">No FTP accounts</p>
                        <p className="text-sm mt-1 opacity-60">Create an account to enable FTP access on your server</p>
                    </div>
                ) : (
                    <table className="w-full text-sm">
                        <thead>
                            <tr className="border-b border-surface-700/50 text-surface-400 text-xs uppercase tracking-wider">
                                <th className="text-left px-5 py-3 font-medium">Username</th>
                                <th className="text-left px-5 py-3 font-medium">Server</th>
                                <th className="text-left px-5 py-3 font-medium">Home Directory</th>
                                <th className="text-left px-5 py-3 font-medium">Quota</th>
                                <th className="text-left px-5 py-3 font-medium">Status</th>
                                <th className="text-right px-5 py-3 font-medium">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-surface-700/30">
                            {displayed.map(acc => (
                                <tr key={acc.id} className="hover:bg-surface-700/20 transition-colors">
                                    <td className="px-5 py-3.5">
                                        <div className="flex items-center gap-2">
                                            <User className="w-4 h-4 text-nova-400" />
                                            <span className="font-mono text-white">ftp_{acc.username}</span>
                                        </div>
                                    </td>
                                    <td className="px-5 py-3.5 text-surface-300">
                                        <div className="flex items-center gap-1.5">
                                            <Server className="w-3.5 h-3.5 text-surface-400" />
                                            {serverName(acc.server_id)}
                                        </div>
                                    </td>
                                    <td className="px-5 py-3.5">
                                        <div className="flex items-center gap-1.5 text-surface-300 font-mono text-xs">
                                            <Folder className="w-3.5 h-3.5 text-surface-400" />
                                            {acc.home_dir}
                                        </div>
                                    </td>
                                    <td className="px-5 py-3.5 text-surface-300 text-xs">
                                        <div className="flex items-center gap-1.5">
                                            <Database className="w-3.5 h-3.5 text-surface-400" />
                                            {acc.quota_mb >= 1024 ? `${(acc.quota_mb / 1024).toFixed(0)} GB` : `${acc.quota_mb} MB`}
                                        </div>
                                    </td>
                                    <td className="px-5 py-3.5">
                                        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${acc.is_active ? 'bg-success/10 text-success' : 'bg-surface-600/50 text-surface-400'}`}>
                                            <span className={`w-1.5 h-1.5 rounded-full ${acc.is_active ? 'bg-success' : 'bg-surface-400'}`} />
                                            {acc.is_active ? 'Active' : 'Inactive'}
                                        </span>
                                    </td>
                                    <td className="px-5 py-3.5 text-right">
                                        <button
                                            onClick={() => handleDelete(acc)}
                                            className="p-1.5 rounded-lg text-surface-400 hover:text-danger hover:bg-danger/10 transition-colors"
                                            title="Delete account"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>

            {/* Info card */}
            <div className="rounded-xl border border-surface-700/30 bg-surface-800/20 p-4 text-sm text-surface-400">
                <p className="font-medium text-surface-300 mb-1">How it works</p>
                <ul className="space-y-1 text-xs">
                    <li>• A system user <span className="font-mono text-surface-300">ftp_&lt;username&gt;</span> is created on the server via SSH</li>
                    <li>• vsftpd is installed automatically if not present</li>
                    <li>• Accounts are chroot-jailed to their home directory</li>
                    <li>• SFTP access works with the same credentials via SSH port 22</li>
                </ul>
            </div>

            {/* Create Modal */}
            {showCreate && (
                <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
                    <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setShowCreate(false)} />
                    <div className="relative z-10 w-full max-w-md rounded-2xl bg-surface-800 border border-surface-700/50 shadow-2xl p-6">
                        <div className="flex items-center justify-between mb-5">
                            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                                <HardDrive className="w-5 h-5 text-nova-400" /> New FTP Account
                            </h2>
                            <button onClick={() => setShowCreate(false)} className="text-surface-400 hover:text-white transition-colors">
                                <X className="w-5 h-5" />
                            </button>
                        </div>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Server</label>
                                <select
                                    value={form.server_id}
                                    onChange={e => setForm(f => ({ ...f, server_id: e.target.value }))}
                                    className={inputCls + ' bg-surface-700/50'}
                                >
                                    <option value="">Select server…</option>
                                    {servers.map(s => (
                                        <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
                                    ))}
                                </select>
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Username</label>
                                <div className="flex items-center gap-0">
                                    <span className="px-3 py-2.5 rounded-l-xl bg-surface-700 border border-r-0 border-surface-600/50 text-surface-400 text-sm font-mono">ftp_</span>
                                    <input
                                        value={form.username}
                                        onChange={e => setForm(f => ({ ...f, username: e.target.value.replace(/[^a-z0-9_-]/gi, '') }))}
                                        placeholder="johndoe"
                                        className={inputCls + ' rounded-l-none'}
                                    />
                                </div>
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Password</label>
                                <div className="relative">
                                    <input
                                        type={showPass ? 'text' : 'password'}
                                        value={form.password}
                                        onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                                        placeholder="Min 8 characters"
                                        className={inputCls + ' pr-10'}
                                    />
                                    <button
                                        type="button"
                                        onClick={() => setShowPass(p => !p)}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 text-surface-400 hover:text-white"
                                    >
                                        {showPass ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                    </button>
                                </div>
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">Home Directory <span className="text-surface-500">(optional)</span></label>
                                <input
                                    value={form.home_dir}
                                    onChange={e => setForm(f => ({ ...f, home_dir: e.target.value }))}
                                    placeholder={`/var/www/${form.username || 'username'}`}
                                    className={inputCls}
                                />
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-surface-300 mb-1.5">
                                    Disk Quota — <span className="text-nova-400">{form.quota_mb >= 1024 ? `${(form.quota_mb / 1024).toFixed(0)} GB` : `${form.quota_mb} MB`}</span>
                                </label>
                                <input
                                    type="range"
                                    min={256}
                                    max={51200}
                                    step={256}
                                    value={form.quota_mb}
                                    onChange={e => setForm(f => ({ ...f, quota_mb: Number(e.target.value) }))}
                                    className="w-full accent-nova-500"
                                />
                                <div className="flex justify-between text-xs text-surface-500 mt-1">
                                    <span>256 MB</span><span>50 GB</span>
                                </div>
                            </div>

                            {formErr && (
                                <p className="text-danger text-xs flex items-center gap-1.5">
                                    <span className="w-1 h-1 rounded-full bg-danger" /> {formErr}
                                </p>
                            )}

                            <button
                                onClick={handleCreate}
                                disabled={creating}
                                className="w-full py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-60 flex items-center justify-center gap-2"
                            >
                                {creating ? (
                                    <><div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Creating…</>
                                ) : (
                                    <><Plus className="w-4 h-4" /> Create FTP Account</>
                                )}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
