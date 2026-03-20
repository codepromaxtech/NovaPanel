import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Server, Plus, Cpu, HardDrive, MemoryStick, Wifi, X,
    WifiOff, Clock, Trash2, Loader, Terminal, Search,
    ChevronDown, Puzzle, Check,
} from 'lucide-react';
import { serverService } from '../services/servers';
import { modulesService, ModuleInfo } from '../services/modules';
import { useToast } from '../components/ui/ToastProvider';

const osOptions = [
    {
        category: 'Debian-based', items: [
            { id: 'Ubuntu 24.04', label: 'Ubuntu 24.04 LTS', logo: '🟠' },
            { id: 'Ubuntu 22.04', label: 'Ubuntu 22.04 LTS', logo: '🟠' },
            { id: 'Ubuntu 20.04', label: 'Ubuntu 20.04 LTS', logo: '🟠' },
            { id: 'Debian 12', label: 'Debian 12 (Bookworm)', logo: '🔴' },
            { id: 'Debian 11', label: 'Debian 11 (Bullseye)', logo: '🔴' },
        ]
    },
    {
        category: 'RHEL-based', items: [
            { id: 'CentOS Stream 9', label: 'CentOS Stream 9', logo: '🟣' },
            { id: 'CentOS Stream 8', label: 'CentOS Stream 8', logo: '🟣' },
            { id: 'AlmaLinux 9', label: 'AlmaLinux 9', logo: '🔵' },
            { id: 'AlmaLinux 8', label: 'AlmaLinux 8', logo: '🔵' },
            { id: 'Rocky Linux 9', label: 'Rocky Linux 9', logo: '🟢' },
            { id: 'Rocky Linux 8', label: 'Rocky Linux 8', logo: '🟢' },
            { id: 'RHEL 9', label: 'RHEL 9', logo: '🔴' },
            { id: 'RHEL 8', label: 'RHEL 8', logo: '🔴' },
            { id: 'Oracle Linux 9', label: 'Oracle Linux 9', logo: '🟤' },
            { id: 'Oracle Linux 8', label: 'Oracle Linux 8', logo: '🟤' },
            { id: 'Fedora 39', label: 'Fedora 39', logo: '🔵' },
            { id: 'Fedora 38', label: 'Fedora 38', logo: '🔵' },
        ]
    },
    {
        category: 'Other', items: [
            { id: 'openSUSE 15', label: 'openSUSE Leap 15', logo: '🟢' },
            { id: 'SLES 15', label: 'SUSE Enterprise 15', logo: '🟢' },
            { id: 'Arch Linux', label: 'Arch Linux', logo: '🔵' },
            { id: 'Alpine 3.19', label: 'Alpine Linux 3.19', logo: '🔵' },
            { id: 'FreeBSD 14', label: 'FreeBSD 14', logo: '🔴' },
            { id: 'Amazon Linux 2023', label: 'Amazon Linux 2023', logo: '🟡' },
            { id: 'Amazon Linux 2', label: 'Amazon Linux 2', logo: '🟡' },
        ]
    },
];

const allOs = osOptions.flatMap(g => g.items);

export default function Servers() {
    const toast = useToast();
    const navigate = useNavigate();
    const [servers, setServers] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAdd, setShowAdd] = useState(false);
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState('');
    const [osSearch, setOsSearch] = useState('');
    const [showOsDropdown, setShowOsDropdown] = useState(false);
    const [newServer, setNewServer] = useState({
        name: '', hostname: '', ip_address: '', port: 22, os: 'Ubuntu 24.04', role: 'worker',
        ssh_user: 'root', auth_method: 'key', ssh_key: '', ssh_password: ''
    });
    const [serverModules, setServerModules] = useState<Record<string, string[]>>({});
    const [availableModules, setAvailableModules] = useState<ModuleInfo[]>([]);
    const [modulePickerServer, setModulePickerServer] = useState<string | null>(null);
    const [selectedModules, setSelectedModules] = useState<string[]>([]);

    const loadServers = async () => {
        setLoading(true);
        try {
            const resp = await serverService.list();
            const list = resp.data || [];
            setServers(list);
            if (list.length > 0) {
                modulesService.listAvailable().then(setAvailableModules).catch(() => { });
                const mods: Record<string, string[]> = {};
                await Promise.all(list.map(async (s: any) => {
                    try {
                        const m = await modulesService.listModules(s.id);
                        mods[s.id] = (m || []).map((x: any) => x.module);
                    } catch { mods[s.id] = []; }
                }));
                setServerModules(mods);
            }
        } catch {
            setServers([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { loadServers(); }, []);

    const handleCreate = async () => {
        if (!newServer.name || !newServer.hostname || !newServer.ip_address) {
            setError('Name, hostname, and IP address are required.');
            return;
        }
        setCreating(true);
        setError('');
        try {
            await serverService.create(newServer);
            toast.success('Server added successfully');
            setShowAdd(false);
            setNewServer({ name: '', hostname: '', ip_address: '', port: 22, os: 'Ubuntu 24.04', role: 'worker', ssh_user: 'root', auth_method: 'key', ssh_key: '', ssh_password: '' });
            loadServers();
        } catch (err: any) {
            setError(err?.response?.data?.error || 'Failed to create server');
        } finally {
            setCreating(false);
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Delete this server?')) return;
        try {
            await serverService.remove(id);
            toast.success('Server deleted');
            loadServers();
        } catch {
            toast.error('Failed to delete server');
        }
    };

    const openModulePicker = (serverId: string) => {
        setModulePickerServer(serverId);
        setSelectedModules(serverModules[serverId] || []);
    };

    const saveModules = async () => {
        if (!modulePickerServer) return;
        try {
            await modulesService.setModules(modulePickerServer, selectedModules);
            toast.success('Modules updated');
            setServerModules(prev => ({ ...prev, [modulePickerServer]: selectedModules }));
            setModulePickerServer(null);
        } catch { toast.error('Failed to update modules'); }
    };

    const statusColor = (s: string) => s === 'active' ? 'text-green-400' : s === 'pending' ? 'text-yellow-400' : 'text-red-400';
    const statusBg = (s: string) => s === 'active' ? 'bg-green-500/10' : s === 'pending' ? 'bg-yellow-500/10' : 'bg-red-500/10';

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Server className="w-7 h-7 text-nova-500" /> Servers
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">{servers.length} server{servers.length !== 1 ? 's' : ''} managed</p>
                </div>
                <button onClick={() => setShowAdd(true)}
                    className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                    <Plus className="w-4 h-4" /> Add Server
                </button>
            </div>

            {loading ? (
                <div className="flex items-center justify-center py-20">
                    <Loader className="w-6 h-6 text-nova-500 animate-spin" />
                </div>
            ) : servers.length === 0 ? (
                <div className="glass-card rounded-2xl flex flex-col items-center justify-center py-20 text-surface-200/40">
                    <Server className="w-16 h-16 mb-4 opacity-40" />
                    <p className="text-lg font-medium mb-2">No servers yet</p>
                    <p className="text-sm mb-6">Add your first server to get started</p>
                    <button onClick={() => setShowAdd(true)}
                        className="px-6 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg transition-all flex items-center gap-2">
                        <Plus className="w-4 h-4" /> Add Server
                    </button>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
                    {servers.map((server: any) => {
                        const mods = serverModules[server.id] || [];
                        return (
                            <div key={server.id} className="glass-card rounded-xl p-5 group hover:border-surface-600/50 transition-all">
                                <div className="flex items-start justify-between mb-4">
                                    <div className="flex items-center gap-3">
                                        <div className="p-2.5 rounded-lg bg-nova-500/10">
                                            <Server className="w-5 h-5 text-nova-400" />
                                        </div>
                                        <div>
                                            <h3 className="text-sm font-semibold text-white">{server.name}</h3>
                                            <p className="text-xs text-surface-200/40">{server.ip_address}:{server.port}</p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium ${statusBg(server.status)} ${statusColor(server.status)}`}>
                                            {server.status}
                                        </span>
                                        <button onClick={() => handleDelete(server.id)}
                                            className="p-1.5 rounded-lg hover:bg-danger/10 text-surface-200/20 hover:text-danger transition-colors opacity-0 group-hover:opacity-100">
                                            <Trash2 className="w-3.5 h-3.5" />
                                        </button>
                                    </div>
                                </div>

                                <div className="grid grid-cols-3 gap-2 mb-3">
                                    <div className="bg-surface-800/40 rounded-lg px-2.5 py-2 text-center">
                                        <Cpu className="w-3.5 h-3.5 mx-auto mb-1 text-blue-400" />
                                        <p className="text-[10px] text-surface-200/30">CPU</p>
                                        <p className="text-xs font-medium text-white">--</p>
                                    </div>
                                    <div className="bg-surface-800/40 rounded-lg px-2.5 py-2 text-center">
                                        <MemoryStick className="w-3.5 h-3.5 mx-auto mb-1 text-green-400" />
                                        <p className="text-[10px] text-surface-200/30">RAM</p>
                                        <p className="text-xs font-medium text-white">--</p>
                                    </div>
                                    <div className="bg-surface-800/40 rounded-lg px-2.5 py-2 text-center">
                                        <HardDrive className="w-3.5 h-3.5 mx-auto mb-1 text-purple-400" />
                                        <p className="text-[10px] text-surface-200/30">Disk</p>
                                        <p className="text-xs font-medium text-white">--</p>
                                    </div>
                                </div>

                                {/* Module chips */}
                                <div className="flex flex-wrap gap-1 mb-3">
                                    {mods.slice(0, 4).map(m => (
                                        <span key={m} className="text-[9px] px-1.5 py-0.5 rounded bg-nova-500/10 text-nova-400 font-medium">{m}</span>
                                    ))}
                                    {mods.length > 4 && <span className="text-[9px] px-1.5 py-0.5 rounded bg-white/5 text-surface-200/40">+{mods.length - 4}</span>}
                                    <button onClick={() => openModulePicker(server.id)}
                                        className="text-[9px] px-1.5 py-0.5 rounded border border-dashed border-surface-600/50 text-surface-200/30 hover:text-nova-400 hover:border-nova-500/30 flex items-center gap-0.5 transition-colors">
                                        <Puzzle className="w-2.5 h-2.5" /> Modules
                                    </button>
                                </div>

                                <div className="flex items-center justify-between pt-3 border-t border-white/5">
                                    <div className="flex items-center gap-2 text-xs text-surface-200/30">
                                        {server.agent_status === 'connected' ? <Wifi className="w-3.5 h-3.5 text-green-400" /> : <WifiOff className="w-3.5 h-3.5 text-red-400" />}
                                        <span>{server.os}</span>
                                    </div>
                                    <button onClick={() => navigate(`/servers/${server.id}/terminal`)}
                                        className="flex items-center gap-1.5 text-xs text-surface-200/30 hover:text-nova-400 transition-colors px-2 py-1 rounded-lg hover:bg-nova-500/10">
                                        <Terminal className="w-3.5 h-3.5" /> Terminal
                                    </button>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            {/* Add Server Modal */}
            {showAdd && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAdd(false)}>
                    <div className="glass-card rounded-2xl p-8 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto custom-scrollbar animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-bold text-white">Add Server</h2>
                            <button onClick={() => setShowAdd(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        {error && (
                            <div className="mb-4 p-3 rounded-xl border border-danger/30 bg-danger/10 text-danger text-sm">{error}</div>
                        )}
                        <div className="space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Server Name</label>
                                    <input value={newServer.name} onChange={e => setNewServer({ ...newServer, name: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                        placeholder="production-01" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Hostname</label>
                                    <input value={newServer.hostname} onChange={e => setNewServer({ ...newServer, hostname: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                        placeholder="server.example.com" />
                                </div>
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">IP Address</label>
                                    <input value={newServer.ip_address} onChange={e => setNewServer({ ...newServer, ip_address: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                        placeholder="0.0.0.0" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">SSH Port</label>
                                    <input type="number" value={newServer.port} onChange={e => setNewServer({ ...newServer, port: parseInt(e.target.value) || 22 })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                                </div>
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                {/* OS Dropdown */}
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Operating System</label>
                                    <div className="relative">
                                        <button type="button" onClick={() => setShowOsDropdown(!showOsDropdown)}
                                            className="w-full flex items-center justify-between px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30">
                                            <div className="flex items-center gap-2">
                                                <span>{allOs.find(o => o.id === newServer.os)?.logo}</span>
                                                <span className="truncate">{allOs.find(o => o.id === newServer.os)?.label || 'Select OS'}</span>
                                            </div>
                                            <ChevronDown className={`w-4 h-4 text-surface-200/50 transition-transform flex-shrink-0 ${showOsDropdown ? 'rotate-180' : ''}`} />
                                        </button>
                                        {showOsDropdown && (
                                            <div className="absolute top-full left-0 right-0 mt-2 p-2 rounded-xl border border-surface-700/50 bg-surface-900/95 backdrop-blur-xl shadow-2xl z-[70]">
                                                <div className="relative mb-2">
                                                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-surface-200/40" />
                                                    <input type="text" placeholder="Search OS..." value={osSearch} onChange={e => setOsSearch(e.target.value)}
                                                        className="w-full pl-9 pr-4 py-2 rounded-lg bg-surface-800/50 border border-surface-700/50 text-white text-xs focus:outline-none focus:ring-1 focus:ring-nova-500/50"
                                                        onClick={e => e.stopPropagation()} autoFocus />
                                                </div>
                                                <div className="max-h-48 overflow-y-auto pr-1 custom-scrollbar space-y-2">
                                                    {osOptions.map(group => {
                                                        const filtered = group.items.filter(os => os.label.toLowerCase().includes(osSearch.toLowerCase()));
                                                        if (!filtered.length) return null;
                                                        return (
                                                            <div key={group.category}>
                                                                <p className="text-[10px] font-semibold text-surface-200/40 uppercase tracking-widest mb-1 px-2">{group.category}</p>
                                                                <div className="space-y-0.5">
                                                                    {filtered.map(os => (
                                                                        <button key={os.id} type="button" onClick={() => { setNewServer({ ...newServer, os: os.id }); setShowOsDropdown(false); setOsSearch(''); }}
                                                                            className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-xs transition-colors ${newServer.os === os.id ? 'bg-nova-500/20 text-white' : 'text-surface-200 hover:bg-surface-800 hover:text-white'}`}>
                                                                            <span className="flex items-center gap-2">{os.logo} {os.label}</span>
                                                                            {newServer.os === os.id && <div className="w-1.5 h-1.5 rounded-full bg-nova-500" />}
                                                                        </button>
                                                                    ))}
                                                                </div>
                                                            </div>
                                                        );
                                                    })}
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                </div>
                                {/* Role  */}
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Role</label>
                                    <select value={newServer.role} onChange={e => setNewServer({ ...newServer, role: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                        <option value="worker">Worker — Runs web apps & databases</option>
                                        <option value="master">Master — Panel host only</option>
                                        <option value="database">Database — Dedicated DB node</option>
                                    </select>
                                </div>
                            </div>

                            {/* SSH Auth Section */}
                            <div className="space-y-4 pt-2">
                                <div className="grid grid-cols-2 gap-4">
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">SSH Username</label>
                                        <input value={newServer.ssh_user} onChange={e => setNewServer({ ...newServer, ssh_user: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                            placeholder="root" />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Auth Method</label>
                                        <div className="flex gap-2">
                                            {[{ id: 'key', label: '🔑 SSH Key' }, { id: 'password', label: '🔒 Password' }].map(m => (
                                                <button key={m.id} type="button" onClick={() => setNewServer({ ...newServer, auth_method: m.id })}
                                                    className={`flex-1 py-2.5 rounded-xl text-sm font-medium transition-all ${newServer.auth_method === m.id ? 'bg-nova-500/20 text-nova-400 border border-nova-500/30' : 'bg-white/5 text-surface-200/50 border border-white/10 hover:bg-white/10'}`}>
                                                    {m.label}
                                                </button>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                                {newServer.auth_method === 'key' ? (
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5 flex items-center justify-between">
                                            <span>SSH Private Key</span>
                                            <span className="text-xs text-surface-200/40 font-normal">PEM format</span>
                                        </label>
                                        <textarea value={newServer.ssh_key} onChange={e => setNewServer({ ...newServer, ssh_key: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-xs font-mono focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20 whitespace-pre custom-scrollbar"
                                            placeholder={"-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"} rows={4} />
                                    </div>
                                ) : (
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">SSH Password</label>
                                        <input type="password" value={newServer.ssh_password} onChange={e => setNewServer({ ...newServer, ssh_password: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                            placeholder="Enter SSH password" />
                                    </div>
                                )}
                            </div>

                            <div className="flex gap-3 pt-2">
                                <button onClick={() => setShowAdd(false)}
                                    className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                                    Cancel
                                </button>
                                <button onClick={handleCreate} disabled={creating}
                                    className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all flex items-center justify-center gap-2 disabled:opacity-50">
                                    {creating ? <Loader className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                                    {creating ? 'Creating...' : 'Add Server'}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* Module Picker Modal */}
            {modulePickerServer && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setModulePickerServer(null)}>
                    <div className="glass-card rounded-2xl p-6 w-full max-w-md mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-5">
                            <h3 className="text-lg font-semibold text-white">Server Modules</h3>
                            <button onClick={() => setModulePickerServer(null)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        <div className="space-y-2 max-h-80 overflow-y-auto custom-scrollbar">
                            {availableModules.map(m => (
                                <label key={m.id} className={`flex items-center gap-3 p-3 rounded-xl border cursor-pointer transition-colors ${selectedModules.includes(m.id) ? 'border-nova-500/50 bg-nova-500/10' : 'border-surface-700/30 hover:bg-surface-800'}`}>
                                    <input type="checkbox" checked={selectedModules.includes(m.id)}
                                        onChange={() => setSelectedModules(prev => prev.includes(m.id) ? prev.filter(x => x !== m.id) : [...prev, m.id])}
                                        className="w-4 h-4 rounded border-surface-600 bg-surface-800 text-nova-500 focus:ring-nova-500" />
                                    <span className="text-base">{m.icon}</span>
                                    <div>
                                        <p className="text-sm font-medium text-white">{m.label}</p>
                                        <p className="text-[10px] text-surface-200/40">{m.category}</p>
                                    </div>
                                </label>
                            ))}
                        </div>
                        <div className="flex gap-3 mt-5">
                            <button onClick={() => setModulePickerServer(null)}
                                className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5">Cancel</button>
                            <button onClick={saveModules}
                                className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg flex items-center justify-center gap-2">
                                <Check className="w-4 h-4" /> Save Modules
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
