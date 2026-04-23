import { useEffect, useState, useRef } from 'react';
import ReactDOM from 'react-dom';
import { useNavigate } from 'react-router-dom';
import {
    Server, Plus, Cpu, HardDrive, MemoryStick, Wifi, X,
    WifiOff, Clock, Trash2, Loader, Terminal, Search,
    ChevronDown, Puzzle, Check, Pencil, Plug, Upload,
    AlertCircle, CheckCircle2,
} from 'lucide-react';
import { serverService } from '../services/servers';
import { modulesService, ModuleInfo } from '../services/modules';
import { phpService, PHPVersion } from '../services/php';
import { useToast } from '../components/ui/ToastProvider';
import { useModalLock } from '../hooks/useModalLock';

const PHP_VERSIONS = ['7.4', '8.0', '8.1', '8.2', '8.3', '8.4', '8.5'];

const osOptions = [
    {
        category: 'Debian-based', items: [
            { id: 'Ubuntu 24.04', label: 'Ubuntu 24.04 LTS', logo: '🟠' },
            { id: 'Ubuntu 22.04', label: 'Ubuntu 22.04 LTS', logo: '🟠' },
            { id: 'Ubuntu 20.04', label: 'Ubuntu 20.04 LTS', logo: '🟠' },
            { id: 'Debian 12', label: 'Debian 12 Bookworm', logo: '🔴' },
            { id: 'Debian 11', label: 'Debian 11 Bullseye', logo: '🔴' },
        ]
    },
    {
        category: 'RHEL-based', items: [
            { id: 'AlmaLinux 9', label: 'AlmaLinux 9', logo: '🔵' },
            { id: 'AlmaLinux 8', label: 'AlmaLinux 8', logo: '🔵' },
            { id: 'Rocky Linux 9', label: 'Rocky Linux 9', logo: '🟢' },
            { id: 'Rocky Linux 8', label: 'Rocky Linux 8', logo: '🟢' },
            { id: 'CentOS Stream 9', label: 'CentOS Stream 9', logo: '🟣' },
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
            { id: 'Arch Linux', label: 'Arch Linux (Rolling)', logo: '🔵' },
            { id: 'Alpine 3.19', label: 'Alpine Linux 3.19', logo: '🔵' },
            { id: 'FreeBSD 14', label: 'FreeBSD 14', logo: '🔴' },
            { id: 'Amazon Linux 2023', label: 'Amazon Linux 2023', logo: '🟡' },
            { id: 'Amazon Linux 2', label: 'Amazon Linux 2', logo: '🟡' },
        ]
    },
];

const allOs = osOptions.flatMap(g => g.items);

function OsDropdown({ currentOs, onSelect }: { currentOs: string; onSelect: (id: string) => void }) {
    const [open, setOpen] = useState(false);
    const [search, setSearch] = useState('');
    const btnRef = useRef<HTMLButtonElement>(null);
    const portalRef = useRef<HTMLDivElement>(null);
    const [rect, setRect] = useState<DOMRect | null>(null);

    const toggle = () => {
        if (!open && btnRef.current) setRect(btnRef.current.getBoundingClientRect());
        setOpen(o => !o);
    };

    useEffect(() => {
        if (!open) return;
        const handler = (e: MouseEvent) => {
            const t = e.target as Node;
            if (!btnRef.current?.contains(t) && !portalRef.current?.contains(t)) {
                setOpen(false);
                setSearch('');
            }
        };
        document.addEventListener('mousedown', handler);
        return () => document.removeEventListener('mousedown', handler);
    }, [open]);

    const selected = allOs.find(o => o.id === currentOs);

    return (
        <>
            <button type="button" ref={btnRef} onClick={toggle}
                className="w-full flex items-center justify-between px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30">
                <div className="flex items-center gap-2">
                    <span>{selected?.logo}</span>
                    <span className="truncate">{selected?.label || 'Select OS'}</span>
                </div>
                <ChevronDown className={`w-4 h-4 text-surface-200/50 transition-transform flex-shrink-0 ${open ? 'rotate-180' : ''}`} />
            </button>
            {open && rect && ReactDOM.createPortal(
                <div ref={portalRef}
                    className="fixed z-[200] p-2 rounded-xl border border-surface-700/50 bg-surface-900/95 backdrop-blur-xl shadow-2xl"
                    style={{ left: rect.left, top: rect.bottom + 6, width: rect.width }}>
                    <div className="relative mb-2">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-surface-200/40" />
                        <input type="text" placeholder="Search OS..." value={search} onChange={e => setSearch(e.target.value)}
                            className="w-full pl-9 pr-4 py-2 rounded-lg bg-surface-800/50 border border-surface-700/50 text-white text-xs focus:outline-none focus:ring-1 focus:ring-nova-500/50"
                            autoFocus onClick={e => e.stopPropagation()} />
                    </div>
                    <div className="max-h-48 overflow-y-auto pr-1 custom-scrollbar space-y-2">
                        {osOptions.map(group => {
                            const filtered = group.items.filter(o => o.label.toLowerCase().includes(search.toLowerCase()));
                            if (!filtered.length) return null;
                            return (
                                <div key={group.category}>
                                    <p className="text-[10px] font-semibold text-surface-200/40 uppercase tracking-widest mb-1 px-2">{group.category}</p>
                                    <div className="space-y-0.5">
                                        {filtered.map(os => (
                                            <button key={os.id} type="button"
                                                onClick={() => { onSelect(os.id); setOpen(false); setSearch(''); }}
                                                className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-xs transition-colors ${currentOs === os.id ? 'bg-nova-500/20 text-white' : 'text-surface-200 hover:bg-surface-800 hover:text-white'}`}>
                                                <span className="flex items-center gap-2">{os.logo} {os.label}</span>
                                                {currentOs === os.id && <div className="w-1.5 h-1.5 rounded-full bg-nova-500" />}
                                            </button>
                                        ))}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </div>,
                document.body
            )}
        </>
    );
}

const emptyServer = {
    name: '', hostname: '', ip_address: '', port: 22, os: 'Ubuntu 24.04', role: 'worker',
    ssh_user: 'root', auth_method: 'password', ssh_key: '', ssh_password: '', modules: [] as string[],
    connect_type: 'ssh' as 'ssh' | 'cloudflare',
    cf_hostname: '', cf_client_id: '', cf_client_secret: '',
};

const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/;
const ipv6Regex = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/;
const isValidIP = (ip: string) => ipv4Regex.test(ip) || ipv6Regex.test(ip);

export default function Servers() {
    const toast = useToast();
    const navigate = useNavigate();
    const [servers, setServers] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAdd, setShowAdd] = useState(false);
    useModalLock(showAdd);
    const [creating, setCreating] = useState(false);
    const [error, setError] = useState('');
    const [formData, setFormData] = useState({ ...emptyServer });
    const [serverModules, setServerModules] = useState<Record<string, string[]>>({});
    const [availableModules, setAvailableModules] = useState<ModuleInfo[]>([]);
    const [modulePickerServer, setModulePickerServer] = useState<string | null>(null);
    const [selectedModules, setSelectedModules] = useState<string[]>([]);
    const [serverMetrics, setServerMetrics] = useState<Record<string, any>>({});

    // Test Connection state
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; output: string } | null>(null);

    // Edit modal state
    const [editServer, setEditServer] = useState<any | null>(null);
    const [editData, setEditData] = useState({ ...emptyServer });
    const [saving, setSaving] = useState(false);
    const [editError, setEditError] = useState('');

    // Create form step (1 = details, 2 = modules)
    const [step, setStep] = useState(1);

    // PHP Version Manager
    const [phpServer, setPHPServer] = useState<string | null>(null);
    const [phpVersions, setPHPVersions] = useState<PHPVersion[]>([]);
    const [phpLoading, setPHPLoading] = useState(false);
    const [phpInstalling, setPHPInstalling] = useState<string | null>(null);

    // SSH key file upload ref
    const keyFileRef = useRef<HTMLInputElement>(null);
    const editKeyFileRef = useRef<HTMLInputElement>(null);

    const loadServers = async () => {
        setLoading(true);
        try {
            const resp = await serverService.list();
            const list = resp.data || [];
            setServers(list);
            if (list.length > 0) {
                modulesService.listAvailable().then(setAvailableModules).catch(() => { });
                const mods: Record<string, string[]> = {};
                const metrics: Record<string, any> = {};
                await Promise.all(list.map(async (s: any) => {
                    try {
                        const m = await modulesService.listModules(s.id);
                        mods[s.id] = (m || []).map((x: any) => x.module);
                    } catch { mods[s.id] = []; }
                    try {
                        const met = await serverService.getLatestMetrics(s.id);
                        if (met.available) metrics[s.id] = met.metrics;
                    } catch { /* no metrics */ }
                }));
                setServerModules(mods);
                setServerMetrics(metrics);
            }
        } catch {
            setServers([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { loadServers(); }, []);

    // Load available modules for the creation form
    useEffect(() => {
        modulesService.listAvailable().then(setAvailableModules).catch(() => { });
    }, []);

    const handleTestConnection = async () => {
        if (formData.connect_type === 'cloudflare' && !formData.cf_hostname) {
            setError('CF Hostname is required to test Cloudflare connection'); return;
        }
        if (formData.connect_type === 'ssh' && !formData.ip_address) {
            setError('IP address is required to test SSH connection'); return;
        }
        if (formData.connect_type === 'ssh' && formData.ip_address && !isValidIP(formData.ip_address)) {
            setError('Please enter a valid IPv4 or IPv6 address'); return;
        }
        setTesting(true);
        setTestResult(null);
        setError('');
        try {
            const result = await serverService.testConnection({
                ip_address: formData.ip_address,
                port: formData.port,
                ssh_user: formData.ssh_user,
                ssh_key: formData.ssh_key,
                ssh_password: formData.ssh_password,
                auth_method: formData.auth_method,
                connect_type: formData.connect_type,
                cf_hostname: formData.cf_hostname,
                cf_client_id: formData.cf_client_id,
                cf_client_secret: formData.cf_client_secret,
            });
            setTestResult(result);
            toast.success('Connection successful!');
        } catch (err: any) {
            setTestResult({ success: false, output: err?.response?.data?.error || 'Connection failed' });
            toast.error('Connection failed');
        } finally {
            setTesting(false);
        }
    };

    const handleCreate = async () => {
        if (!formData.name || !formData.hostname) {
            setError('Name and hostname are required.');
            return;
        }
        if (formData.connect_type === 'ssh' && !formData.ip_address) {
            setError('IP address is required for SSH connections.');
            return;
        }
        if (formData.connect_type === 'cloudflare' && !formData.cf_hostname) {
            setError('CF Hostname is required for Cloudflare connections.');
            return;
        }
        if (formData.connect_type === 'ssh' && formData.ip_address && !isValidIP(formData.ip_address)) {
            setError('Please enter a valid IPv4 or IPv6 address');
            return;
        }
        setCreating(true);
        setError('');
        try {
            await serverService.create(formData);
            toast.success('Server added successfully! Provisioning will start shortly.');
            setShowAdd(false);
            setFormData({ ...emptyServer });
            setStep(1);
            setTestResult(null);
            loadServers();
        } catch (err: any) {
            setError(err?.response?.data?.error || 'Failed to create server');
        } finally {
            setCreating(false);
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Delete this server? This cannot be undone.')) return;
        try {
            await serverService.remove(id);
            toast.success('Server deleted');
            loadServers();
        } catch {
            toast.error('Failed to delete server');
        }
    };

    const openEdit = (server: any) => {
        setEditServer(server);
        setEditData({
            name: server.name || '',
            hostname: server.hostname || '',
            ip_address: server.ip_address || '',
            port: server.port || 22,
            os: server.os || 'Ubuntu 24.04',
            role: server.role || 'worker',
            ssh_user: server.ssh_user || 'root',
            auth_method: server.auth_method || 'password',
            ssh_key: server.ssh_key || '',
            ssh_password: '',
            modules: [],
            connect_type: (server.connect_type || 'ssh') as 'ssh' | 'cloudflare',
            cf_hostname: server.cf_hostname || '',
            cf_client_id: '',
            cf_client_secret: '',
        });
        setEditError('');
    };

    const handleUpdate = async () => {
        if (!editServer) return;
        if (editData.connect_type === 'ssh' && editData.ip_address && !isValidIP(editData.ip_address)) {
            setEditError('Please enter a valid IPv4 or IPv6 address');
            return;
        }
        setSaving(true);
        setEditError('');
        try {
            await serverService.update(editServer.id, editData);
            toast.success('Server updated');
            setEditServer(null);
            loadServers();
        } catch (err: any) {
            setEditError(err?.response?.data?.error || 'Failed to update server');
        } finally {
            setSaving(false);
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

    const openPHPManager = async (serverId: string) => {
        setPHPServer(serverId);
        setPHPLoading(true);
        try {
            const versions = await phpService.list(serverId);
            setPHPVersions(versions);
        } catch { setPHPVersions([]); } finally { setPHPLoading(false); }
    };

    const handleInstallPHP = async (version: string) => {
        if (!phpServer) return;
        setPHPInstalling(version);
        try {
            const pv = await phpService.install(phpServer, version);
            setPHPVersions(prev => [...prev.filter(v => v.version !== version), pv]);
            toast.success(`PHP ${version} installation started`);
        } catch (err: any) {
            toast.error(err?.response?.data?.error || `Failed to install PHP ${version}`);
        } finally { setPHPInstalling(null); }
    };

    const handleSetDefaultPHP = async (version: string) => {
        if (!phpServer) return;
        try {
            await phpService.setDefault(phpServer, version);
            setPHPVersions(prev => prev.map(v => ({ ...v, is_default: v.version === version })));
            toast.success(`PHP ${version} set as default`);
        } catch { toast.error('Failed to set default'); }
    };

    const handleUninstallPHP = async (version: string) => {
        if (!phpServer || !confirm(`Uninstall PHP ${version}?`)) return;
        try {
            await phpService.uninstall(phpServer, version);
            setPHPVersions(prev => prev.filter(v => v.version !== version));
            toast.success(`PHP ${version} uninstall started`);
        } catch { toast.error('Failed to uninstall'); }
    };

    const handleKeyFileUpload = (e: React.ChangeEvent<HTMLInputElement>, target: 'create' | 'edit') => {
        const file = e.target.files?.[0];
        if (!file) return;
        const reader = new FileReader();
        reader.onload = () => {
            const content = reader.result as string;
            if (target === 'create') setFormData(prev => ({ ...prev, ssh_key: content }));
            else setEditData(prev => ({ ...prev, ssh_key: content }));
        };
        reader.readAsText(file);
    };

    const statusColor = (s: string) => s === 'active' ? 'text-green-400' : s === 'pending' ? 'text-yellow-400' : 'text-red-400';
    const statusBg = (s: string) => s === 'active' ? 'bg-green-500/10' : s === 'pending' ? 'bg-yellow-500/10' : 'bg-red-500/10';

    const formatMetric = (val: number | undefined, suffix: string) => val !== undefined ? `${Math.round(val)}${suffix}` : '--';

    // ─── Reusable SSH Auth Section ───
    const renderSSHAuth = (data: typeof emptyServer, setData: (fn: (prev: typeof emptyServer) => typeof emptyServer) => void, fileRef: React.RefObject<HTMLInputElement>) => (
        <div className="space-y-4 pt-2">
            <div className="grid grid-cols-2 gap-4">
                <div>
                    <label className="block text-sm font-medium text-surface-200 mb-1.5">SSH Username</label>
                    <input value={data.ssh_user} onChange={e => setData(prev => ({ ...prev, ssh_user: e.target.value }))}
                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                        placeholder="root" />
                </div>
                <div>
                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Auth Method</label>
                    <div className="flex gap-2">
                        {[{ id: 'key', label: '🔑 SSH Key' }, { id: 'password', label: '🔒 Password' }].map(m => (
                            <button key={m.id} type="button" onClick={() => setData(prev => ({ ...prev, auth_method: m.id }))}
                                className={`flex-1 py-2.5 rounded-xl text-sm font-medium transition-all ${data.auth_method === m.id ? 'bg-nova-500/20 text-nova-400 border border-nova-500/30' : 'bg-white/5 text-surface-200/50 border border-white/10 hover:bg-white/10'}`}>
                                {m.label}
                            </button>
                        ))}
                    </div>
                </div>
            </div>
            {data.auth_method === 'key' ? (
                <div>
                    <label className="block text-sm font-medium text-surface-200 mb-1.5 flex items-center justify-between">
                        <span>SSH Private Key</span>
                        <button type="button" onClick={() => fileRef.current?.click()}
                            className="flex items-center gap-1 text-xs text-nova-400 hover:text-nova-300 font-normal transition-colors">
                            <Upload className="w-3 h-3" /> Upload .pem
                        </button>
                    </label>
                    <input type="file" ref={fileRef} className="hidden" accept=".pem,.key,.pub,.ppk,*"
                        onChange={e => handleKeyFileUpload(e, data === formData ? 'create' : 'edit')} />
                    <textarea value={data.ssh_key} onChange={e => setData(prev => ({ ...prev, ssh_key: e.target.value }))}
                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-xs font-mono focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20 whitespace-pre custom-scrollbar"
                        placeholder={"-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"} rows={4} />
                </div>
            ) : (
                <div>
                    <label className="block text-sm font-medium text-surface-200 mb-1.5">SSH Password</label>
                    <input type="password" value={data.ssh_password} onChange={e => setData(prev => ({ ...prev, ssh_password: e.target.value }))}
                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                        placeholder="Enter SSH password" />
                </div>
            )}
        </div>
    );

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Server className="w-7 h-7 text-nova-500" /> Servers
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">{servers.length} server{servers.length !== 1 ? 's' : ''} managed</p>
                </div>
                <button onClick={() => { setShowAdd(true); setStep(1); setTestResult(null); setError(''); setFormData({ ...emptyServer }); }}
                    className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                    <Plus className="w-4 h-4" /> Add Server
                </button>
            </div>

            {loading ? (
                <div className="flex items-center justify-center py-20">
                    <Loader className="w-6 h-6 text-nova-400 animate-spin" />
                </div>
            ) : servers.length === 0 ? (
                <div className="glass-card rounded-2xl flex flex-col items-center justify-center py-20 text-surface-200/40">
                    <Server className="w-16 h-16 mb-4 opacity-40" />
                    <p className="text-lg font-medium mb-2">No servers yet</p>
                    <p className="text-sm mb-6">Add your first server to get started</p>
                    <button onClick={() => { setShowAdd(true); setStep(1); }}
                        className="px-6 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg transition-all flex items-center gap-2">
                        <Plus className="w-4 h-4" /> Add Server
                    </button>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
                    {servers.map((server: any) => {
                        const mods = serverModules[server.id] || [];
                        const met = serverMetrics[server.id];
                        return (
                            <div key={server.id} className="glass-card rounded-xl p-5 group hover:border-surface-600/50 transition-all">
                                <div className="flex items-start justify-between mb-4">
                                    <div className="flex items-center gap-3">
                                        <div className="p-2.5 rounded-lg bg-nova-500/10">
                                            <Server className="w-5 h-5 text-nova-400" />
                                        </div>
                                        <div>
                                            <h3 className="text-sm font-semibold text-white">{server.name}</h3>
                                            <p className="text-xs text-surface-200/40">
                                                {server.connect_type === 'cloudflare'
                                                    ? `🌐 ${server.cf_hostname || server.hostname}`
                                                    : `${server.ip_address}:${server.port}`}
                                            </p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium ${statusBg(server.status)} ${statusColor(server.status)}`}>
                                            {server.status}
                                        </span>
                                        <button onClick={() => openEdit(server)}
                                            className="p-1.5 rounded-lg hover:bg-nova-500/10 text-surface-200/40 hover:text-nova-400 transition-colors opacity-0 group-hover:opacity-100">
                                            <Pencil className="w-3.5 h-3.5" />
                                        </button>
                                        <button onClick={() => handleDelete(server.id)}
                                            className="p-1.5 rounded-lg hover:bg-danger/10 text-surface-200/40 hover:text-danger transition-colors opacity-0 group-hover:opacity-100">
                                            <Trash2 className="w-3.5 h-3.5" />
                                        </button>
                                    </div>
                                </div>

                                {/* Live Metrics */}
                                <div className="grid grid-cols-3 gap-2 mb-3">
                                    <div className="bg-surface-800/40 rounded-lg px-2.5 py-2 text-center">
                                        <Cpu className="w-3.5 h-3.5 mx-auto mb-1 text-blue-400" />
                                        <p className="text-[10px] text-surface-200/50">CPU</p>
                                        <p className="text-xs font-medium text-white">{formatMetric(met?.cpu_percent, '%')}</p>
                                    </div>
                                    <div className="bg-surface-800/40 rounded-lg px-2.5 py-2 text-center">
                                        <MemoryStick className="w-3.5 h-3.5 mx-auto mb-1 text-green-400" />
                                        <p className="text-[10px] text-surface-200/50">RAM</p>
                                        <p className="text-xs font-medium text-white">
                                            {met ? `${Math.round(met.ram_used_mb)}/${Math.round(met.ram_total_mb)}M` : '--'}
                                        </p>
                                    </div>
                                    <div className="bg-surface-800/40 rounded-lg px-2.5 py-2 text-center">
                                        <HardDrive className="w-3.5 h-3.5 mx-auto mb-1 text-purple-400" />
                                        <p className="text-[10px] text-surface-200/50">Disk</p>
                                        <p className="text-xs font-medium text-white">
                                            {met ? `${met.disk_used_gb.toFixed(1)}/${met.disk_total_gb.toFixed(1)}G` : '--'}
                                        </p>
                                    </div>
                                </div>

                                {/* Module chips */}
                                <div className="flex flex-wrap gap-1 mb-3">
                                    {mods.slice(0, 4).map(m => (
                                        <span key={m} className="text-[9px] px-1.5 py-0.5 rounded bg-nova-500/10 text-nova-400 font-medium">{m}</span>
                                    ))}
                                    {mods.length > 4 && <span className="text-[9px] px-1.5 py-0.5 rounded bg-white/5 text-surface-200/40">+{mods.length - 4}</span>}
                                    <button onClick={() => openModulePicker(server.id)}
                                        className="text-[9px] px-1.5 py-0.5 rounded border border-dashed border-surface-600/50 text-surface-200/50 hover:text-nova-400 hover:border-nova-500/30 flex items-center gap-0.5 transition-colors">
                                        <Puzzle className="w-2.5 h-2.5" /> Modules
                                    </button>
                                </div>

                                <div className="flex items-center justify-between pt-3 border-t border-white/5">
                                    <div className="flex items-center gap-2 text-xs text-surface-200/50">
                                        {server.agent_status === 'connected' ? <Wifi className="w-3.5 h-3.5 text-green-400" /> : <WifiOff className="w-3.5 h-3.5 text-red-400" />}
                                        <span>{server.os}</span>
                                    </div>
                                    <div className="flex items-center gap-1">
                                        <button onClick={() => openPHPManager(server.id)}
                                            className="flex items-center gap-1.5 text-xs text-surface-200/50 hover:text-purple-400 transition-colors px-2 py-1 rounded-lg hover:bg-purple-500/10"
                                            title="PHP Version Manager">
                                            🐘 PHP
                                        </button>
                                        <button onClick={() => navigate(`/servers/${server.id}/terminal`)}
                                            className="flex items-center gap-1.5 text-xs text-surface-200/50 hover:text-nova-400 transition-colors px-2 py-1 rounded-lg hover:bg-nova-500/10">
                                            <Terminal className="w-3.5 h-3.5" /> Terminal
                                        </button>
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            {/* ═══════ Add Server Modal (Multi-Step) ═══════ */}
            {showAdd && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAdd(false)}>
                    <div className="glass-card rounded-2xl p-8 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto custom-scrollbar animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-bold text-white">Add Server — Step {step}/2</h2>
                            <button onClick={() => setShowAdd(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>

                        {/* Step indicator */}
                        <div className="flex gap-2 mb-6">
                            <div className={`flex-1 h-1 rounded-full transition-colors ${step >= 1 ? 'bg-nova-500' : 'bg-surface-700'}`} />
                            <div className={`flex-1 h-1 rounded-full transition-colors ${step >= 2 ? 'bg-nova-500' : 'bg-surface-700'}`} />
                        </div>

                        {error && (
                            <div className="mb-4 p-3 rounded-xl border border-danger/30 bg-danger/10 text-danger text-sm flex items-center gap-2">
                                <AlertCircle className="w-4 h-4 flex-shrink-0" /> {error}
                            </div>
                        )}

                        {step === 1 && (
                            <div className="space-y-4">
                                <div className="grid grid-cols-2 gap-4">
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Server Name</label>
                                        <input value={formData.name} onChange={e => setFormData({ ...formData, name: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                            placeholder="production-01" />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Hostname</label>
                                        <input value={formData.hostname} onChange={e => setFormData({ ...formData, hostname: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                            placeholder="server.example.com" />
                                    </div>
                                </div>

                                {/* Connection Type Toggle */}
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Connection Type</label>
                                    <div className="flex gap-2">
                                        {([
                                            { id: 'ssh', label: '🔐 Direct SSH' },
                                            { id: 'cloudflare', label: '🌐 Cloudflare Access' },
                                        ] as const).map(ct => (
                                            <button key={ct.id} type="button"
                                                onClick={() => setFormData(prev => ({ ...prev, connect_type: ct.id }))}
                                                className={`flex-1 py-2.5 rounded-xl text-sm font-medium transition-all ${formData.connect_type === ct.id ? 'bg-nova-500/20 text-nova-400 border border-nova-500/30' : 'bg-white/5 text-surface-200/50 border border-white/10 hover:bg-white/10'}`}>
                                                {ct.label}
                                            </button>
                                        ))}
                                    </div>
                                </div>

                                {formData.connect_type === 'ssh' ? (
                                    <div className="grid grid-cols-2 gap-4">
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">IP Address</label>
                                            <input value={formData.ip_address} onChange={e => { setFormData({ ...formData, ip_address: e.target.value }); setTestResult(null); }}
                                                className={`w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 placeholder:text-surface-200/20 ${formData.ip_address && !isValidIP(formData.ip_address) ? 'focus:ring-red-500/30 border-red-500/30' : 'focus:ring-nova-500/30'}`}
                                                placeholder="0.0.0.0" />
                                            {formData.ip_address && !isValidIP(formData.ip_address) && (
                                                <p className="text-[10px] text-red-400 mt-1">Invalid IP address format</p>
                                            )}
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">SSH Port</label>
                                            <input type="number" value={formData.port} onChange={e => setFormData({ ...formData, port: parseInt(e.target.value) || 22 })}
                                                className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                                        </div>
                                    </div>
                                ) : (
                                    <div className="space-y-3 p-3 rounded-xl border border-orange-500/20 bg-orange-500/5">
                                        <div className="flex items-start gap-2 text-xs text-orange-300/80 mb-1">
                                            <span>⚠️</span>
                                            <span>Cloudflare Access SSH requires <code>cloudflared</code> on the panel host and a Service Token with <strong>SSH</strong> access to your tunnel.</span>
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">CF Tunnel Hostname</label>
                                            <input value={formData.cf_hostname} onChange={e => setFormData({ ...formData, cf_hostname: e.target.value })}
                                                className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                                placeholder="ssh.example.com" />
                                        </div>
                                        <div className="grid grid-cols-2 gap-3">
                                            <div>
                                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Client ID</label>
                                                <input value={formData.cf_client_id} onChange={e => setFormData({ ...formData, cf_client_id: e.target.value })}
                                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                                    placeholder="xxxx.access" />
                                            </div>
                                            <div>
                                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Client Secret</label>
                                                <input type="password" value={formData.cf_client_secret} onChange={e => setFormData({ ...formData, cf_client_secret: e.target.value })}
                                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                                    placeholder="••••••••" />
                                            </div>
                                        </div>
                                    </div>
                                )}

                                <div className="grid grid-cols-2 gap-4">
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Operating System</label>
                                        <OsDropdown currentOs={formData.os} onSelect={(id) => setFormData({ ...formData, os: id })} />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Role</label>
                                        <select value={formData.role} onChange={e => setFormData({ ...formData, role: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                            <option value="worker">Worker — Runs web apps & databases</option>
                                            <option value="master">Master — Panel host only</option>
                                            <option value="database">Database — Dedicated DB node</option>
                                        </select>
                                    </div>
                                </div>

                                {renderSSHAuth(formData, setFormData as any, keyFileRef as any)}

                                {/* Test Connection Button */}
                                <div className="pt-2">
                                    <button onClick={handleTestConnection} disabled={testing || (formData.connect_type === 'ssh' ? !formData.ip_address : !formData.cf_hostname)}
                                        className="w-full py-2.5 rounded-xl border border-nova-500/30 text-nova-400 text-sm font-medium hover:bg-nova-500/10 transition-all flex items-center justify-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed">
                                        {testing ? <Loader className="w-4 h-4 animate-spin" /> : <Plug className="w-4 h-4" />}
                                        {testing ? 'Testing...' : `Test ${formData.connect_type === 'cloudflare' ? 'Cloudflare' : 'SSH'} Connection`}
                                    </button>
                                    {testResult && (
                                        <div className={`mt-3 p-3 rounded-xl border text-xs font-mono whitespace-pre-wrap max-h-32 overflow-y-auto custom-scrollbar ${testResult.success ? 'border-green-500/30 bg-green-500/10 text-green-300' : 'border-red-500/30 bg-red-500/10 text-red-300'}`}>
                                            <div className="flex items-center gap-2 mb-1 font-sans font-medium text-sm">
                                                {testResult.success ? <CheckCircle2 className="w-4 h-4 text-green-400" /> : <AlertCircle className="w-4 h-4 text-red-400" />}
                                                {testResult.success ? 'Connection Successful' : 'Connection Failed'}
                                            </div>
                                            {testResult.output}
                                        </div>
                                    )}
                                </div>

                                <div className="flex gap-3 pt-2">
                                    <button onClick={() => setShowAdd(false)}
                                        className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                                        Cancel
                                    </button>
                                    <button onClick={() => { setError(''); setStep(2); }}
                                        className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                                        Next: Select Modules →
                                    </button>
                                </div>
                            </div>
                        )}

                        {step === 2 && (
                            <div className="space-y-4">
                                <p className="text-sm text-surface-200/60">Select modules to auto-install on the server after creation. You can change these later.</p>
                                <div className="space-y-2 max-h-80 overflow-y-auto custom-scrollbar pb-4">
                                    {availableModules.map(m => (
                                        <label key={m.id} className={`flex items-center gap-3 p-3 rounded-xl border cursor-pointer transition-colors ${formData.modules.includes(m.id) ? 'border-nova-500/50 bg-nova-500/10' : 'border-surface-700/30 hover:bg-surface-800'}`}>
                                            <input type="checkbox" checked={formData.modules.includes(m.id)}
                                                onChange={() => setFormData(prev => ({ ...prev, modules: prev.modules.includes(m.id) ? prev.modules.filter(x => x !== m.id) : [...prev.modules, m.id] }))}
                                                className="w-4 h-4 rounded border-surface-600 bg-surface-800 text-nova-500 focus:ring-nova-500" />
                                            <span className="text-base">{m.icon}</span>
                                            <div>
                                                <p className="text-sm font-medium text-white">{m.label}</p>
                                                <p className="text-[10px] text-surface-200/40">{m.category}</p>
                                            </div>
                                        </label>
                                    ))}
                                    {availableModules.length === 0 && (
                                        <p className="text-sm text-surface-200/40 text-center py-6">No modules available. You can add them later.</p>
                                    )}
                                </div>

                                <div className="flex gap-3 pt-2">
                                    <button onClick={() => setStep(1)}
                                        className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                                        ← Back
                                    </button>
                                    <button onClick={handleCreate} disabled={creating}
                                        className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all flex items-center justify-center gap-2 disabled:opacity-50">
                                        {creating ? <Loader className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                                        {creating ? 'Creating...' : `Add Server${formData.modules.length > 0 ? ` + ${formData.modules.length} modules` : ''}`}
                                    </button>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* ═══════ Edit Server Modal ═══════ */}
            {editServer && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setEditServer(null)}>
                    <div className="glass-card rounded-2xl p-8 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto custom-scrollbar animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-bold text-white flex items-center gap-2"><Pencil className="w-5 h-5 text-nova-400" /> Edit Server</h2>
                            <button onClick={() => setEditServer(null)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        {editError && (
                            <div className="mb-4 p-3 rounded-xl border border-danger/30 bg-danger/10 text-danger text-sm flex items-center gap-2">
                                <AlertCircle className="w-4 h-4 flex-shrink-0" /> {editError}
                            </div>
                        )}
                        <div className="space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Server Name</label>
                                    <input value={editData.name} onChange={e => setEditData({ ...editData, name: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Hostname</label>
                                    <input value={editData.hostname} onChange={e => setEditData({ ...editData, hostname: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                                </div>
                            </div>

                            {/* Connection Type */}
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Connection Type</label>
                                <div className="flex gap-2">
                                    {([
                                        { id: 'ssh', label: '🔐 Direct SSH' },
                                        { id: 'cloudflare', label: '🌐 Cloudflare Access' },
                                    ] as const).map(ct => (
                                        <button key={ct.id} type="button"
                                            onClick={() => setEditData(prev => ({ ...prev, connect_type: ct.id }))}
                                            className={`flex-1 py-2.5 rounded-xl text-sm font-medium transition-all ${editData.connect_type === ct.id ? 'bg-nova-500/20 text-nova-400 border border-nova-500/30' : 'bg-white/5 text-surface-200/50 border border-white/10 hover:bg-white/10'}`}>
                                            {ct.label}
                                        </button>
                                    ))}
                                </div>
                            </div>

                            {editData.connect_type === 'ssh' ? (
                                <div className="grid grid-cols-2 gap-4">
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">IP Address</label>
                                        <input value={editData.ip_address} onChange={e => setEditData({ ...editData, ip_address: e.target.value })}
                                            className={`w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 ${editData.ip_address && !isValidIP(editData.ip_address) ? 'focus:ring-red-500/30' : 'focus:ring-nova-500/30'}`} />
                                        {editData.ip_address && !isValidIP(editData.ip_address) && (
                                            <p className="text-[10px] text-red-400 mt-1">Invalid IP address format</p>
                                        )}
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">SSH Port</label>
                                        <input type="number" value={editData.port} onChange={e => setEditData({ ...editData, port: parseInt(e.target.value) || 22 })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                                    </div>
                                </div>
                            ) : (
                                <div className="space-y-3 p-3 rounded-xl border border-orange-500/20 bg-orange-500/5">
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200 mb-1.5">CF Tunnel Hostname</label>
                                        <input value={editData.cf_hostname} onChange={e => setEditData({ ...editData, cf_hostname: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                            placeholder="ssh.example.com" />
                                    </div>
                                    <div className="grid grid-cols-2 gap-3">
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">Client ID</label>
                                            <input value={editData.cf_client_id} onChange={e => setEditData({ ...editData, cf_client_id: e.target.value })}
                                                className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                                placeholder="leave blank to keep" />
                                        </div>
                                        <div>
                                            <label className="block text-sm font-medium text-surface-200 mb-1.5">Client Secret</label>
                                            <input type="password" value={editData.cf_client_secret} onChange={e => setEditData({ ...editData, cf_client_secret: e.target.value })}
                                                className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                                placeholder="leave blank to keep" />
                                        </div>
                                    </div>
                                </div>
                            )}

                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Operating System</label>
                                    <OsDropdown currentOs={editData.os} onSelect={(id) => setEditData({ ...editData, os: id })} />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Role</label>
                                    <select value={editData.role} onChange={e => setEditData({ ...editData, role: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                        <option value="worker">Worker</option>
                                        <option value="master">Master</option>
                                        <option value="database">Database</option>
                                    </select>
                                </div>
                            </div>

                            {renderSSHAuth(editData, setEditData as any, editKeyFileRef as any)}

                            <div className="flex gap-3 pt-2">
                                <button onClick={() => setEditServer(null)}
                                    className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                                    Cancel
                                </button>
                                <button onClick={handleUpdate} disabled={saving}
                                    className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all flex items-center justify-center gap-2 disabled:opacity-50">
                                    {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
                                    {saving ? 'Saving...' : 'Save Changes'}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* ═══════ Module Picker Modal ═══════ */}
            {modulePickerServer && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setModulePickerServer(null)}>
                    <div className="glass-card rounded-2xl p-6 w-full max-w-md mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-5">
                            <h3 className="text-lg font-semibold text-white">Server Modules</h3>
                            <button onClick={() => setModulePickerServer(null)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        <div className="space-y-2 max-h-80 overflow-y-auto custom-scrollbar pb-4">
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

            {/* ═══════ PHP Version Manager Modal ═══════ */}
            {phpServer && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setPHPServer(null)}>
                    <div className="glass-card rounded-2xl p-6 w-full max-w-lg mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-5">
                            <h3 className="text-lg font-semibold text-white flex items-center gap-2">🐘 PHP Version Manager</h3>
                            <button onClick={() => setPHPServer(null)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>

                        {phpLoading ? (
                            <div className="flex items-center justify-center py-10"><Loader className="w-5 h-5 text-nova-400 animate-spin" /></div>
                        ) : (
                            <div className="space-y-2">
                                {PHP_VERSIONS.map(ver => {
                                    const installed = phpVersions.find(v => v.version === ver);
                                    const isInstalling = installed?.status === 'installing' || phpInstalling === ver;
                                    return (
                                        <div key={ver} className="flex items-center justify-between p-3 rounded-xl border border-white/[0.06] bg-white/[0.02] hover:bg-white/[0.04] transition-colors">
                                            <div className="flex items-center gap-3">
                                                <span className="font-mono text-sm font-semibold text-white">PHP {ver}</span>
                                                {installed?.is_default && (
                                                    <span className="text-[10px] px-1.5 py-0.5 rounded bg-nova-500/20 text-nova-400 font-medium">default</span>
                                                )}
                                                {isInstalling && (
                                                    <span className="text-[10px] text-yellow-400 flex items-center gap-1"><Loader className="w-3 h-3 animate-spin" /> installing</span>
                                                )}
                                                {installed && !isInstalling && (
                                                    <span className="text-[10px] text-green-400 flex items-center gap-1"><Check className="w-3 h-3" /> installed</span>
                                                )}
                                            </div>
                                            <div className="flex items-center gap-1">
                                                {!installed ? (
                                                    <button
                                                        onClick={() => handleInstallPHP(ver)}
                                                        disabled={phpInstalling !== null}
                                                        className="text-xs px-3 py-1.5 rounded-lg bg-nova-500/10 text-nova-400 hover:bg-nova-500/20 transition-colors disabled:opacity-40"
                                                    >
                                                        Install
                                                    </button>
                                                ) : (
                                                    <>
                                                        {!installed.is_default && (
                                                            <button
                                                                onClick={() => handleSetDefaultPHP(ver)}
                                                                className="text-xs px-2 py-1.5 rounded-lg text-surface-200/40 hover:text-nova-400 hover:bg-nova-500/10 transition-colors"
                                                            >
                                                                Set Default
                                                            </button>
                                                        )}
                                                        <button
                                                            onClick={() => handleUninstallPHP(ver)}
                                                            className="p-1.5 rounded-lg text-surface-200/50 hover:text-danger hover:bg-danger/10 transition-colors"
                                                        >
                                                            <Trash2 className="w-3.5 h-3.5" />
                                                        </button>
                                                    </>
                                                )}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        )}

                        <button onClick={() => setPHPServer(null)}
                            className="w-full mt-5 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                            Close
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
