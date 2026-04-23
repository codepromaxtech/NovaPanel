import { useEffect, useState, useCallback } from 'react';
import {
    Box, Play, Square, RotateCw, Trash2, FileText,
    Download, HardDrive, Network, Layers, PlusCircle,
    Search, RefreshCw, X, Copy, Edit3, Cpu, MemoryStick,
    Terminal, Activity, Zap, Info, LayoutGrid, ArrowUpDown,
    Server, Gauge, Pause, FolderOpen, Upload, Hammer,
    Clock, Skull, Save
} from 'lucide-react';
import { dockerService } from '../services/docker';
import { useAuthStore } from '../store/authStore';

type Tab = 'containers' | 'images' | 'volumes' | 'networks' | 'stacks' | 'templates' | 'system' | 'events';

interface ContainerItem {
    id: string; name: string; image: string; state: string; status: string;
    ports: { private: number; public: number; type: string }[];
    created: number; labels?: Record<string, string>;
}
interface ImageItem { id: string; tags: string[]; size: number; created: number; }
interface VolumeItem { name: string; driver: string; mountpoint: string; created_at: string; }
interface NetworkItem { id: string; name: string; driver: string; scope: string; subnet?: string; gateway?: string; }
interface StackItem { id: string; name: string; status: string; services: number; compose_yaml?: string; created_at: string; }
interface TemplateItem { id: string; title: string; description: string; category: string; logo: string; image: string; ports?: Record<string, string>; env?: { name: string; label: string; default: string }[]; }
interface SystemInfo { docker_version: string; api_version: string; os: string; arch: string; kernel: string; cpus: number; memory: number; storage_driver: string; root_dir: string; containers: number; images: number; runtime: string; }
interface ContainerStats { cpu_percent: number; mem_usage: number; mem_limit: number; mem_percent: number; net_rx: number; net_tx: number; block_read: number; block_write: number; pids: number; }

function fmtBytes(b: number) {
    if (b < 1024) return b + ' B';
    if (b < 1048576) return (b / 1024).toFixed(1) + ' KB';
    if (b < 1073741824) return (b / 1048576).toFixed(1) + ' MB';
    return (b / 1073741824).toFixed(2) + ' GB';
}
function timeAgo(epoch: number) {
    const ms = Date.now() - epoch * 1000;
    if (ms < 60000) return 'just now';
    if (ms < 3600000) return Math.floor(ms / 60000) + 'm ago';
    if (ms < 86400000) return Math.floor(ms / 3600000) + 'h ago';
    return Math.floor(ms / 86400000) + 'd ago';
}

const stateColors: Record<string, string> = {
    running: 'text-emerald-400 bg-emerald-400/10 border-emerald-400/30',
    exited: 'text-red-400 bg-red-400/10 border-red-400/30',
    paused: 'text-amber-400 bg-amber-400/10 border-amber-400/30',
    created: 'text-blue-400 bg-blue-400/10 border-blue-400/30',
    restarting: 'text-amber-400 bg-amber-400/10 border-amber-400/30',
};

export default function Docker() {
    const [tab, setTab] = useState<Tab>('containers');
    const [containers, setContainers] = useState<ContainerItem[]>([]);
    const [images, setImages] = useState<ImageItem[]>([]);
    const [volumes, setVolumes] = useState<VolumeItem[]>([]);
    const [networks, setNetworks] = useState<NetworkItem[]>([]);
    const [stacks, setStacks] = useState<StackItem[]>([]);
    const [templates, setTemplates] = useState<TemplateItem[]>([]);
    const [sysInfo, setSysInfo] = useState<SystemInfo | null>(null);
    const [stats, setStats] = useState({ containers: 0, running: 0, stopped: 0, images: 0, volumes: 0, networks: 0 });
    const [containerStatsMap, setContainerStatsMap] = useState<Record<string, ContainerStats>>({});
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');
    const [logModal, setLogModal] = useState<{ id: string; name: string; logs: string } | null>(null);
    const [inspectModal, setInspectModal] = useState<{ id: string; name: string; data: string } | null>(null);
    const [showCreate, setShowCreate] = useState(false);
    const { token } = useAuthStore();

    // Forms
    const [newContainer, setNewContainer] = useState({ name: '', image: '', env: '', restart: 'unless-stopped', ports: '', memory_mb: 0, cpus: 0 });
    const [pullImg, setPullImg] = useState('');
    const [pulling, setPulling] = useState(false);
    const [newVol, setNewVol] = useState('');
    const [newNet, setNewNet] = useState('');
    const [newStack, setNewStack] = useState({ name: '', yaml: '' });
    const [templateDeploy, setTemplateDeploy] = useState<TemplateItem | null>(null);
    const [templateEnv, setTemplateEnv] = useState<Record<string, string>>({});
    const [templateName, setTemplateName] = useState('');
    const [events, setEvents] = useState<{ type: string; action: string; actor: string; time: number }[]>([]);
    const [fileModal, setFileModal] = useState<{ id: string; name: string; path: string; files: { name: string; path: string; size: number; is_dir: boolean; mod_time: number; mode: string }[] } | null>(null);
    const [buildForm, setBuildForm] = useState({ dockerfile: '', tag: '' });
    const [building, setBuilding] = useState(false);

    const load = useCallback(async () => {
        setLoading(true);
        try {
            const [c, i, v, n, s, st, t] = await Promise.all([
                dockerService.listContainers(true),
                dockerService.listImages(),
                dockerService.listVolumes(),
                dockerService.listNetworks(),
                dockerService.listStacks(),
                dockerService.getStats(),
                dockerService.listTemplates(),
            ]);
            setContainers(c); setImages(i); setVolumes(v); setNetworks(n); setStacks(s); setStats(st); setTemplates(t);
        } catch { /* silent */ }
        setLoading(false);
    }, []);

    // Load container stats for running containers
    useEffect(() => {
        if (tab !== 'containers') return;
        const running = containers.filter(c => c.state === 'running');
        if (running.length === 0) return;
        const loadStats = async () => {
            const map: Record<string, ContainerStats> = {};
            await Promise.all(running.map(async c => {
                try { map[c.id] = await dockerService.containerStats(c.id); } catch { /* skip */ }
            }));
            setContainerStatsMap(map);
        };
        loadStats();
        const t = setInterval(loadStats, 5000);
        return () => clearInterval(t);
    }, [containers, tab]);

    // Load system info
    useEffect(() => {
        if (tab === 'system') {
            dockerService.systemInfo().then(setSysInfo).catch(() => { });
        }
        if (tab === 'events') {
            dockerService.getEvents('24h').then(setEvents).catch(() => { });
        }
    }, [tab]);

    useEffect(() => { load(); const t = setInterval(load, 10000); return () => clearInterval(t); }, [load]);

    const act = async (fn: () => Promise<unknown>) => { try { await fn(); load(); } catch { /* silent */ } };

    const viewLogs = async (id: string, name: string) => {
        const logs = await dockerService.containerLogs(id, '500');
        setLogModal({ id, name, logs });
    };

    const viewInspect = async (id: string, name: string) => {
        const data = await dockerService.inspectContainer(id);
        setInspectModal({ id, name, data: JSON.stringify(data, null, 2) });
    };

    const openExec = (id: string) => {
        const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${wsProto}//${window.location.hostname}:8080/api/v1/docker/containers/${id}/exec?shell=/bin/sh&token=${token}`;
        window.open(`/servers/${id}/terminal?ws=${encodeURIComponent(url)}&title=Container+Exec`, '_blank');
    };

    const renameContainer = async (id: string) => {
        const name = prompt('New container name:');
        if (name) await act(() => dockerService.renameContainer(id, name));
    };

    const duplicateContainer = async (id: string) => {
        const name = prompt('Name for the clone:');
        if (name) await act(() => dockerService.duplicateContainer(id, name));
    };

    const commitContainer = async (id: string) => {
        const repo = prompt('Repository name for the image (e.g. myapp):');
        if (!repo) return;
        const tag = prompt('Tag (default: latest):') || 'latest';
        await act(() => dockerService.commitContainer(id, repo, tag, 'Committed from NovaPanel'));
    };

    const browseContainer = async (id: string, name: string, dirPath = '/') => {
        try {
            const resp = await dockerService.browseFiles(id, dirPath);
            setFileModal({ id, name, path: resp.path, files: resp.data || [] });
        } catch { /* */ }
    };

    const createContainer = async () => {
        const envArr = newContainer.env ? newContainer.env.split('\n').filter(Boolean) : [];
        const ports: Record<string, string> = {};
        newContainer.ports.split('\n').filter(Boolean).forEach(line => {
            const [host, cont] = line.split(':');
            if (host && cont) ports[`${cont}/tcp`] = host;
        });
        await act(() => dockerService.createContainer({
            name: newContainer.name, image: newContainer.image, env: envArr, restart: newContainer.restart,
            ports: Object.keys(ports).length > 0 ? ports : undefined,
            memory_mb: newContainer.memory_mb || undefined,
            cpus: newContainer.cpus || undefined,
        }));
        setNewContainer({ name: '', image: '', env: '', restart: 'unless-stopped', ports: '', memory_mb: 0, cpus: 0 });
        setShowCreate(false);
    };

    const deployTemplateAction = async () => {
        if (!templateDeploy) return;
        await act(() => dockerService.deployTemplate(templateDeploy.id, templateEnv, templateName || undefined));
        setTemplateDeploy(null); setTemplateEnv({}); setTemplateName('');
    };

    const prune = async () => {
        if (!confirm('This will remove all stopped containers, unused images, volumes, and networks. Continue?')) return;
        const result = await dockerService.pruneSystem();
        alert(`Pruned: ${result.containers_deleted} containers, ${result.images_deleted} images, ${result.volumes_deleted} volumes, ${result.networks_deleted} networks. Reclaimed: ${fmtBytes(result.space_reclaimed)}`);
        load();
    };

    const tabs: { key: Tab; label: string; icon: typeof Box; count?: number }[] = [
        { key: 'containers', label: 'Containers', icon: Box, count: stats.containers },
        { key: 'images', label: 'Images', icon: Layers, count: stats.images },
        { key: 'volumes', label: 'Volumes', icon: HardDrive, count: stats.volumes },
        { key: 'networks', label: 'Networks', icon: Network, count: stats.networks },
        { key: 'stacks', label: 'Stacks', icon: LayoutGrid, count: stacks.length },
        { key: 'templates', label: 'Templates', icon: Zap, count: templates.length },
        { key: 'system', label: 'System', icon: Server },
        { key: 'events', label: 'Events', icon: Activity },
    ];

    const filtered = <T,>(items: T[]) =>
        items.filter(x => !search || JSON.stringify(x).toLowerCase().includes(search.toLowerCase()));

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-2">
                        <Box className="w-7 h-7 text-blue-400" /> Docker Management
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">
                        {stats.running} running · {stats.stopped} stopped · {stats.images} images · {stats.volumes} volumes · {stats.networks} networks
                    </p>
                </div>
                <button onClick={load} className="p-2 rounded-lg bg-surface-700/50 hover:bg-surface-600/50 text-surface-200/60 hover:text-white transition-all">
                    <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
                </button>
            </div>

            {/* Tabs */}
            <div className="flex items-center gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1.5 overflow-x-auto">
                {tabs.map(t => (
                    <button key={t.key} onClick={() => setTab(t.key)}
                        className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all whitespace-nowrap ${tab === t.key ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-surface-200/50 hover:text-white hover:bg-surface-700/50 border border-transparent'
                            }`}>
                        <t.icon className="w-4 h-4" />
                        {t.label}
                        {t.count !== undefined && <span className={`text-xs px-1.5 py-0.5 rounded-full ${tab === t.key ? 'bg-blue-500/20 text-blue-300' : 'bg-surface-700/50 text-surface-200/40'}`}>{t.count}</span>}
                    </button>
                ))}
            </div>

            {/* Search + Actions */}
            <div className="flex items-center gap-3">
                <div className="flex-1 relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-surface-200/50" />
                    <input value={search} onChange={e => setSearch(e.target.value)} placeholder={`Search ${tab}...`}
                        className="w-full pl-10 pr-4 py-2.5 bg-surface-800/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none transition-colors" />
                </div>
                {tab === 'containers' && (
                    <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-blue-600 to-blue-700 text-white rounded-lg text-sm font-medium hover:from-blue-500 hover:to-blue-600 transition-all">
                        <PlusCircle className="w-4 h-4" /> New Container
                    </button>
                )}
            </div>

            {/* Create Container Form */}
            {showCreate && tab === 'containers' && (
                <div className="bg-surface-800/50 border border-blue-500/20 rounded-xl p-5 space-y-4">
                    <h3 className="text-white font-semibold text-sm">Create Container</h3>
                    <div className="grid grid-cols-2 gap-4">
                        <input value={newContainer.name} onChange={e => setNewContainer(p => ({ ...p, name: e.target.value }))} placeholder="Container name"
                            className="px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                        <input value={newContainer.image} onChange={e => setNewContainer(p => ({ ...p, image: e.target.value }))} placeholder="Image (e.g. nginx:latest)"
                            className="px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                    </div>
                    <textarea value={newContainer.ports} onChange={e => setNewContainer(p => ({ ...p, ports: e.target.value }))} placeholder="Port mapping (one per line: hostPort:containerPort)" rows={2}
                        className="w-full px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none font-mono" />
                    <textarea value={newContainer.env} onChange={e => setNewContainer(p => ({ ...p, env: e.target.value }))} placeholder="ENV vars (one per line: KEY=VALUE)" rows={2}
                        className="w-full px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none font-mono" />
                    <div className="flex items-center gap-3 flex-wrap">
                        <select value={newContainer.restart} onChange={e => setNewContainer(p => ({ ...p, restart: e.target.value }))}
                            className="px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none">
                            <option value="">No restart</option>
                            <option value="always">Always</option>
                            <option value="unless-stopped">Unless stopped</option>
                            <option value="on-failure">On failure</option>
                        </select>
                        <div className="flex items-center gap-2">
                            <Cpu className="w-4 h-4 text-surface-200/40" />
                            <input type="number" value={newContainer.cpus || ''} onChange={e => setNewContainer(p => ({ ...p, cpus: parseFloat(e.target.value) || 0 }))} placeholder="CPUs (e.g. 0.5)" step="0.1" min="0"
                                className="w-28 px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                        </div>
                        <div className="flex items-center gap-2">
                            <MemoryStick className="w-4 h-4 text-surface-200/40" />
                            <input type="number" value={newContainer.memory_mb || ''} onChange={e => setNewContainer(p => ({ ...p, memory_mb: parseInt(e.target.value) || 0 }))} placeholder="RAM (MB)" min="0"
                                className="w-28 px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                        </div>
                        <button onClick={createContainer} disabled={!newContainer.name || !newContainer.image}
                            className="px-4 py-2 bg-emerald-600 text-white rounded-lg text-sm font-medium hover:bg-emerald-500 transition-all disabled:opacity-40 disabled:cursor-not-allowed">
                            Create & Start
                        </button>
                    </div>
                </div>
            )}

            {/* ===== CONTAINERS TAB ===== */}
            {tab === 'containers' && (
                <div className="space-y-3">
                    {filtered(containers).map(c => {
                        const s = containerStatsMap[c.id];
                        return (
                            <div key={c.id} className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-600/50 transition-all group">
                                <div className="flex items-center gap-4">
                                    <div className="w-10 h-10 rounded-lg bg-blue-500/10 border border-blue-500/20 flex items-center justify-center shrink-0">
                                        <Box className="w-5 h-5 text-blue-400" />
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2">
                                            <span className="text-white font-medium text-sm truncate">{c.name}</span>
                                            <span className={`text-xs px-2 py-0.5 rounded-full border ${stateColors[c.state] || 'text-gray-400 bg-gray-400/10 border-gray-400/30'}`}>{c.state}</span>
                                        </div>
                                        <div className="flex items-center gap-3 mt-1 text-xs text-surface-200/40">
                                            <span className="font-mono">{c.id}</span>
                                            <span>{c.image}</span>
                                            {c.ports?.filter(p => p.public > 0).map((p, i) => (
                                                <span key={i} className="text-blue-400">{p.public}→{p.private}</span>
                                            ))}
                                            <span>{timeAgo(c.created)}</span>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-1 opacity-60 group-hover:opacity-100 transition-opacity">
                                        {c.state !== 'running' && (
                                            <button onClick={() => act(() => dockerService.startContainer(c.id))} title="Start" className="p-2 rounded-lg hover:bg-emerald-500/20 text-emerald-400"><Play className="w-4 h-4" /></button>
                                        )}
                                        {c.state === 'running' && <>
                                            <button onClick={() => { if (confirm('Stop this container?')) act(() => dockerService.stopContainer(c.id)); }} title="Stop" className="p-2 rounded-lg hover:bg-amber-500/20 text-amber-400"><Square className="w-4 h-4" /></button>
                                            <button onClick={() => act(() => dockerService.pauseContainer(c.id))} title="Pause" className="p-2 rounded-lg hover:bg-yellow-500/20 text-yellow-400"><Pause className="w-4 h-4" /></button>
                                            <button onClick={() => openExec(c.id)} title="Console" className="p-2 rounded-lg hover:bg-cyan-500/20 text-cyan-400"><Terminal className="w-4 h-4" /></button>
                                        </>}
                                        {c.state === 'paused' && <>
                                            <button onClick={() => act(() => dockerService.unpauseContainer(c.id))} title="Unpause" className="p-2 rounded-lg hover:bg-green-500/20 text-green-400"><Play className="w-4 h-4" /></button>
                                            <button onClick={() => { if (confirm('Kill this container? This will forcefully terminate it.')) act(() => dockerService.killContainer(c.id)); }} title="Kill" className="p-2 rounded-lg hover:bg-red-500/20 text-red-400"><Skull className="w-4 h-4" /></button>
                                        </>}
                                        <button onClick={() => act(() => dockerService.restartContainer(c.id))} title="Restart" className="p-2 rounded-lg hover:bg-blue-500/20 text-blue-400"><RotateCw className="w-4 h-4" /></button>
                                        <button onClick={() => viewLogs(c.id, c.name)} title="Logs" className="p-2 rounded-lg hover:bg-purple-500/20 text-purple-400"><FileText className="w-4 h-4" /></button>
                                        <button onClick={() => viewInspect(c.id, c.name)} title="Inspect" className="p-2 rounded-lg hover:bg-sky-500/20 text-sky-400"><Info className="w-4 h-4" /></button>
                                        <button onClick={() => browseContainer(c.id, c.name)} title="Files" className="p-2 rounded-lg hover:bg-indigo-500/20 text-indigo-400"><FolderOpen className="w-4 h-4" /></button>
                                        <button onClick={() => renameContainer(c.id)} title="Rename" className="p-2 rounded-lg hover:bg-orange-500/20 text-orange-400"><Edit3 className="w-4 h-4" /></button>
                                        <button onClick={() => duplicateContainer(c.id)} title="Duplicate" className="p-2 rounded-lg hover:bg-teal-500/20 text-teal-400"><Copy className="w-4 h-4" /></button>
                                        <button onClick={() => commitContainer(c.id)} title="Commit to Image" className="p-2 rounded-lg hover:bg-lime-500/20 text-lime-400"><Save className="w-4 h-4" /></button>
                                        <button onClick={() => { if (confirm('Remove this container? This action cannot be undone.')) act(() => dockerService.removeContainer(c.id, true)); }} title="Remove" className="p-2 rounded-lg hover:bg-red-500/20 text-red-400"><Trash2 className="w-4 h-4" /></button>
                                    </div>
                                </div>
                                {/* Live stats bar */}
                                {s && c.state === 'running' && (
                                    <div className="mt-3 pt-3 border-t border-surface-700/30 grid grid-cols-4 gap-4">
                                        <div>
                                            <div className="flex items-center justify-between text-xs mb-1"><span className="text-surface-200/40 flex items-center gap-1"><Cpu className="w-3 h-3" /> CPU</span><span className="text-emerald-400">{s.cpu_percent.toFixed(1)}%</span></div>
                                            <div className="h-1.5 bg-surface-700/50 rounded-full overflow-hidden"><div className="h-full bg-emerald-500 rounded-full transition-all" style={{ width: `${Math.min(s.cpu_percent, 100)}%` }} /></div>
                                        </div>
                                        <div>
                                            <div className="flex items-center justify-between text-xs mb-1"><span className="text-surface-200/40 flex items-center gap-1"><MemoryStick className="w-3 h-3" /> RAM</span><span className="text-blue-400">{fmtBytes(s.mem_usage)}</span></div>
                                            <div className="h-1.5 bg-surface-700/50 rounded-full overflow-hidden"><div className="h-full bg-blue-500 rounded-full transition-all" style={{ width: `${Math.min(s.mem_percent, 100)}%` }} /></div>
                                        </div>
                                        <div>
                                            <div className="flex items-center justify-between text-xs mb-1"><span className="text-surface-200/40 flex items-center gap-1"><ArrowUpDown className="w-3 h-3" /> Net</span><span className="text-purple-400">{fmtBytes(s.net_rx)}/{fmtBytes(s.net_tx)}</span></div>
                                            <div className="h-1.5 bg-surface-700/50 rounded-full overflow-hidden"><div className="h-full bg-purple-500 rounded-full transition-all" style={{ width: '30%' }} /></div>
                                        </div>
                                        <div>
                                            <div className="flex items-center justify-between text-xs mb-1"><span className="text-surface-200/40 flex items-center gap-1"><Activity className="w-3 h-3" /> PIDs</span><span className="text-amber-400">{s.pids}</span></div>
                                            <div className="flex items-center gap-2 text-xs text-surface-200/50">IO: {fmtBytes(s.block_read)}R / {fmtBytes(s.block_write)}W</div>
                                        </div>
                                    </div>
                                )}
                            </div>
                        );
                    })}
                    {containers.length === 0 && <p className="text-center text-surface-200/50 py-8">No containers found</p>}
                </div>
            )}

            {/* ===== IMAGES TAB ===== */}
            {tab === 'images' && (
                <div className="space-y-3">
                    <div className="flex items-center gap-3 mb-4">
                        <input value={pullImg} onChange={e => setPullImg(e.target.value)} placeholder="Pull image (e.g. nginx:latest)"
                            className="flex-1 px-3 py-2.5 bg-surface-800/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                        <button onClick={async () => { setPulling(true); await act(() => dockerService.pullImage(pullImg)); setPullImg(''); setPulling(false); }} disabled={!pullImg || pulling}
                            className="flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-blue-600 to-blue-700 text-white rounded-lg text-sm font-medium hover:from-blue-500 hover:to-blue-600 transition-all disabled:opacity-40">
                            <Download className={`w-4 h-4 ${pulling ? 'animate-bounce' : ''}`} /> {pulling ? 'Pulling...' : 'Pull'}
                        </button>
                    </div>
                    {/* Build Image Form */}
                    <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 mb-4 space-y-3">
                        <h3 className="text-white font-semibold text-sm flex items-center gap-2"><Hammer className="w-4 h-4 text-amber-400" /> Build Image from Dockerfile</h3>
                        <textarea value={buildForm.dockerfile} onChange={e => setBuildForm(p => ({ ...p, dockerfile: e.target.value }))} placeholder={'FROM nginx:alpine\nCOPY . /usr/share/nginx/html'} rows={4}
                            className="w-full px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-xs focus:border-blue-500/50 focus:outline-none font-mono" />
                        <div className="flex items-center gap-3">
                            <input value={buildForm.tag} onChange={e => setBuildForm(p => ({ ...p, tag: e.target.value }))} placeholder="Tag (e.g. myapp:latest)"
                                className="flex-1 px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                            <button onClick={async () => { setBuilding(true); await act(() => dockerService.buildImage(buildForm.dockerfile, buildForm.tag)); setBuildForm({ dockerfile: '', tag: '' }); setBuilding(false); }}
                                disabled={!buildForm.dockerfile || !buildForm.tag || building}
                                className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-amber-600 to-amber-700 text-white rounded-lg text-sm font-medium hover:from-amber-500 hover:to-amber-600 transition-all disabled:opacity-40">
                                <Hammer className={`w-4 h-4 ${building ? 'animate-spin' : ''}`} /> {building ? 'Building...' : 'Build'}
                            </button>
                        </div>
                    </div>
                    {filtered(images.map(i => ({ ...i, name: (i.tags?.[0] || i.id) }))).map(img => (
                        <div key={img.id} className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center gap-4 hover:border-surface-600/50 transition-all group">
                            <div className="w-10 h-10 rounded-lg bg-purple-500/10 border border-purple-500/20 flex items-center justify-center">
                                <Layers className="w-5 h-5 text-purple-400" />
                            </div>
                            <div className="flex-1 min-w-0">
                                <span className="text-white font-medium text-sm">{img.tags?.join(', ') || '<none>'}</span>
                                <div className="flex items-center gap-3 mt-1 text-xs text-surface-200/40">
                                    <span className="font-mono">{img.id}</span>
                                    <span>{fmtBytes(img.size)}</span>
                                    <span>{timeAgo(img.created)}</span>
                                </div>
                            </div>
                            <button onClick={async () => { const ref = img.tags?.[0]; if (ref && confirm(`Push ${ref} to registry?`)) await act(() => dockerService.pushImage(ref)); }} title="Push to Registry"
                                className="p-2 rounded-lg hover:bg-blue-500/20 text-blue-400 opacity-60 group-hover:opacity-100 transition-all"><Upload className="w-4 h-4" /></button>
                            <button onClick={() => { if (confirm('Remove this image?')) act(() => dockerService.removeImage(img.id)); }} title="Remove"
                                className="p-2 rounded-lg hover:bg-red-500/20 text-red-400 opacity-60 group-hover:opacity-100 transition-all"><Trash2 className="w-4 h-4" /></button>
                        </div>
                    ))}
                    {images.length === 0 && <p className="text-center text-surface-200/50 py-8">No images found</p>}
                </div>
            )}

            {/* ===== VOLUMES TAB ===== */}
            {tab === 'volumes' && (
                <div className="space-y-3">
                    <div className="flex items-center gap-3 mb-4">
                        <input value={newVol} onChange={e => setNewVol(e.target.value)} placeholder="New volume name"
                            className="flex-1 px-3 py-2.5 bg-surface-800/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                        <button onClick={async () => { await act(() => dockerService.createVolume(newVol)); setNewVol(''); }} disabled={!newVol}
                            className="flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-emerald-600 to-emerald-700 text-white rounded-lg text-sm font-medium hover:from-emerald-500 hover:to-emerald-600 transition-all disabled:opacity-40">
                            <PlusCircle className="w-4 h-4" /> Create
                        </button>
                    </div>
                    {filtered(volumes.map(v => ({ ...v, id: v.name }))).map(vol => (
                        <div key={vol.name} className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center gap-4 hover:border-surface-600/50 transition-all group">
                            <div className="w-10 h-10 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center"><HardDrive className="w-5 h-5 text-emerald-400" /></div>
                            <div className="flex-1 min-w-0">
                                <span className="text-white font-medium text-sm">{vol.name}</span>
                                <div className="flex items-center gap-3 mt-1 text-xs text-surface-200/40">
                                    <span>Driver: {vol.driver}</span><span className="truncate max-w-xs">{vol.mountpoint}</span>
                                </div>
                            </div>
                            <button onClick={() => { if (confirm(`Remove volume "${vol.name}"? All data will be lost.`)) act(() => dockerService.removeVolume(vol.name)); }} title="Remove" className="p-2 rounded-lg hover:bg-red-500/20 text-red-400 opacity-60 group-hover:opacity-100 transition-all"><Trash2 className="w-4 h-4" /></button>
                        </div>
                    ))}
                    {volumes.length === 0 && <p className="text-center text-surface-200/50 py-8">No volumes found</p>}
                </div>
            )}

            {/* ===== NETWORKS TAB ===== */}
            {tab === 'networks' && (
                <div className="space-y-3">
                    <div className="flex items-center gap-3 mb-4">
                        <input value={newNet} onChange={e => setNewNet(e.target.value)} placeholder="New network name"
                            className="flex-1 px-3 py-2.5 bg-surface-800/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                        <button onClick={async () => { await act(() => dockerService.createNetwork(newNet)); setNewNet(''); }} disabled={!newNet}
                            className="flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-cyan-600 to-cyan-700 text-white rounded-lg text-sm font-medium hover:from-cyan-500 hover:to-cyan-600 transition-all disabled:opacity-40">
                            <PlusCircle className="w-4 h-4" /> Create
                        </button>
                    </div>
                    {filtered(networks).map(net => (
                        <div key={net.id} className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center gap-4 hover:border-surface-600/50 transition-all group">
                            <div className="w-10 h-10 rounded-lg bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center"><Network className="w-5 h-5 text-cyan-400" /></div>
                            <div className="flex-1 min-w-0">
                                <span className="text-white font-medium text-sm">{net.name}</span>
                                <div className="flex items-center gap-3 mt-1 text-xs text-surface-200/40">
                                    <span>Driver: {net.driver}</span><span>Scope: {net.scope}</span>
                                    {net.subnet && <span>Subnet: {net.subnet}</span>}{net.gateway && <span>GW: {net.gateway}</span>}
                                </div>
                            </div>
                            {!['bridge', 'host', 'none'].includes(net.name) && (
                                <button onClick={() => { if (confirm(`Remove network "${net.name}"?`)) act(() => dockerService.removeNetwork(net.id)); }} title="Remove" className="p-2 rounded-lg hover:bg-red-500/20 text-red-400 opacity-60 group-hover:opacity-100 transition-all"><Trash2 className="w-4 h-4" /></button>
                            )}
                        </div>
                    ))}
                    {networks.length === 0 && <p className="text-center text-surface-200/50 py-8">No networks found</p>}
                </div>
            )}

            {/* ===== STACKS TAB ===== */}
            {tab === 'stacks' && (
                <div className="space-y-4">
                    <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-5 space-y-4">
                        <h3 className="text-white font-semibold text-sm flex items-center gap-2"><LayoutGrid className="w-4 h-4 text-amber-400" /> Deploy Stack</h3>
                        <input value={newStack.name} onChange={e => setNewStack(p => ({ ...p, name: e.target.value }))} placeholder="Stack name"
                            className="w-full px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                        <textarea value={newStack.yaml} onChange={e => setNewStack(p => ({ ...p, yaml: e.target.value }))} placeholder="Paste docker-compose.yml..." rows={8}
                            className="w-full px-3 py-2 bg-surface-900/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none font-mono" />
                        <button onClick={async () => { await act(() => dockerService.deployStack({ name: newStack.name, compose_yaml: newStack.yaml })); setNewStack({ name: '', yaml: '' }); }}
                            disabled={!newStack.name || !newStack.yaml} className="px-4 py-2 bg-gradient-to-r from-amber-600 to-amber-700 text-white rounded-lg text-sm font-medium hover:from-amber-500 hover:to-amber-600 transition-all disabled:opacity-40">Deploy Stack</button>
                    </div>
                    {stacks.map(s => (
                        <div key={s.id} className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center gap-4 hover:border-surface-600/50 transition-all group">
                            <div className="w-10 h-10 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center"><LayoutGrid className="w-5 h-5 text-amber-400" /></div>
                            <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2">
                                    <span className="text-white font-medium text-sm">{s.name}</span>
                                    <span className="text-xs px-2 py-0.5 rounded-full border text-emerald-400 bg-emerald-400/10 border-emerald-400/30">{s.status}</span>
                                </div>
                                <span className="text-xs text-surface-200/40">{s.services} service{s.services !== 1 ? 's' : ''}</span>
                            </div>
                            <button onClick={() => { if (confirm(`Remove stack "${s.name}"? All associated containers will be stopped and removed.`)) act(() => dockerService.removeStack(s.name)); }} title="Remove" className="p-2 rounded-lg hover:bg-red-500/20 text-red-400 opacity-60 group-hover:opacity-100 transition-all"><Trash2 className="w-4 h-4" /></button>
                        </div>
                    ))}
                    {stacks.length === 0 && <p className="text-center text-surface-200/50 py-4 text-sm">No stacks deployed</p>}
                </div>
            )}

            {/* ===== TEMPLATES TAB ===== */}
            {tab === 'templates' && (
                <div>
                    {/* Categories */}
                    {Array.from(new Set(templates.map(t => t.category))).map(cat => (
                        <div key={cat} className="mb-6">
                            <h3 className="text-white font-semibold text-sm mb-3 flex items-center gap-2">{cat}</h3>
                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
                                {filtered(templates.filter(t => t.category === cat)).map(t => (
                                    <button key={t.id} onClick={() => { setTemplateDeploy(t); setTemplateEnv({}); setTemplateName(t.id); }}
                                        className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 text-left hover:border-blue-500/30 hover:bg-surface-800/80 transition-all group">
                                        <div className="flex items-center gap-3 mb-2">
                                            <span className="text-2xl">{t.logo}</span>
                                            <div>
                                                <p className="text-white font-medium text-sm group-hover:text-blue-400 transition-colors">{t.title}</p>
                                                <p className="text-xs text-surface-200/50 font-mono">{t.image}</p>
                                            </div>
                                        </div>
                                        <p className="text-xs text-surface-200/40 line-clamp-2">{t.description}</p>
                                        {t.ports && <div className="mt-2 flex gap-1 flex-wrap">
                                            {Object.entries(t.ports).map(([k, v]) => (
                                                <span key={k} className="text-xs px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400">{v}→{k.split('/')[0]}</span>
                                            ))}
                                        </div>}
                                    </button>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* ===== SYSTEM TAB ===== */}
            {tab === 'system' && sysInfo && (
                <div className="space-y-4">
                    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                        {[
                            { label: 'Docker', value: sysInfo.docker_version, icon: Box, color: 'blue' },
                            { label: 'API', value: sysInfo.api_version, icon: Server, color: 'cyan' },
                            { label: 'CPUs', value: `${sysInfo.cpus} cores`, icon: Cpu, color: 'emerald' },
                            { label: 'Memory', value: fmtBytes(sysInfo.memory), icon: MemoryStick, color: 'purple' },
                        ].map(s => (
                            <div key={s.label} className={`bg-surface-800/50 border border-surface-700/50 rounded-xl p-4`}>
                                <div className="flex items-center gap-2 mb-1">
                                    <s.icon className={`w-4 h-4 text-${s.color}-400`} /><span className="text-xs text-surface-200/40">{s.label}</span>
                                </div>
                                <p className="text-white font-semibold text-lg">{s.value}</p>
                            </div>
                        ))}
                    </div>
                    <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-5 space-y-3">
                        <h3 className="text-white font-semibold flex items-center gap-2"><Gauge className="w-4 h-4 text-blue-400" /> Engine Details</h3>
                        <div className="grid grid-cols-2 gap-2 text-sm">
                            {[
                                ['OS', sysInfo.os], ['Architecture', sysInfo.arch], ['Kernel', sysInfo.kernel],
                                ['Storage Driver', sysInfo.storage_driver], ['Root Dir', sysInfo.root_dir], ['Runtime', sysInfo.runtime],
                                ['Containers', `${sysInfo.containers}`], ['Images', `${sysInfo.images}`],
                            ].map(([k, v]) => (
                                <div key={k} className="flex items-center justify-between py-1.5 border-b border-surface-700/30">
                                    <span className="text-surface-200/40">{k}</span><span className="text-white font-mono text-xs">{v}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                    <button onClick={prune} className="flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-red-600 to-red-700 text-white rounded-lg text-sm font-medium hover:from-red-500 hover:to-red-600 transition-all">
                        <Trash2 className="w-4 h-4" /> Prune Unused Resources
                    </button>
                </div>
            )}

            {/* ===== EVENTS TAB ===== */}
            {tab === 'events' && (
                <div className="space-y-3">
                    <div className="flex items-center justify-between mb-2">
                        <h3 className="text-white font-semibold flex items-center gap-2"><Activity className="w-4 h-4 text-purple-400" /> Docker Events (last 24h)</h3>
                        <button onClick={() => dockerService.getEvents('24h').then(setEvents)} className="text-xs text-blue-400 hover:underline">Refresh</button>
                    </div>
                    {events.length === 0 ? (
                        <p className="text-center text-surface-200/50 py-8">No events found</p>
                    ) : (
                        <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl overflow-hidden">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="border-b border-surface-700/50 text-surface-200/40 text-xs">
                                        <th className="text-left px-4 py-3">Time</th>
                                        <th className="text-left px-4 py-3">Type</th>
                                        <th className="text-left px-4 py-3">Action</th>
                                        <th className="text-left px-4 py-3">Actor</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {events.slice(0, 100).map((e, i) => (
                                        <tr key={i} className="border-b border-surface-700/20 hover:bg-surface-700/20">
                                            <td className="px-4 py-2.5 text-xs text-surface-200/50 font-mono flex items-center gap-1.5">
                                                <Clock className="w-3 h-3" />
                                                {new Date(e.time * 1000).toLocaleTimeString()}
                                            </td>
                                            <td className="px-4 py-2.5">
                                                <span className="text-xs px-2 py-0.5 rounded-full border border-blue-500/30 bg-blue-500/10 text-blue-400">{e.type}</span>
                                            </td>
                                            <td className="px-4 py-2.5">
                                                <span className={`text-xs px-2 py-0.5 rounded-full border ${e.action.includes('die') || e.action.includes('kill') || e.action.includes('destroy') ? 'border-red-500/30 bg-red-500/10 text-red-400' :
                                                    e.action.includes('start') || e.action.includes('create') ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400' :
                                                        'border-surface-500/30 bg-surface-500/10 text-surface-300'
                                                    }`}>{e.action}</span>
                                            </td>
                                            <td className="px-4 py-2.5 text-xs text-white font-mono truncate max-w-xs">{e.actor}</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>
            )}

            {/* ===== LOG MODAL ===== */}
            {logModal && (
                <div className="fixed inset-0 bg-black/70 z-[60] flex items-center justify-center p-8" onClick={() => setLogModal(null)}>
                    <div className="bg-surface-900 border border-surface-700/50 rounded-2xl w-full max-w-4xl max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between px-5 py-4 border-b border-surface-700/50">
                            <h3 className="text-white font-semibold flex items-center gap-2"><FileText className="w-4 h-4 text-purple-400" /> Logs: {logModal.name}</h3>
                            <button onClick={() => setLogModal(null)} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white"><X className="w-4 h-4" /></button>
                        </div>
                        <pre className="flex-1 overflow-auto p-5 text-xs text-surface-200/70 font-mono whitespace-pre-wrap leading-relaxed">{logModal.logs || 'No logs available'}</pre>
                    </div>
                </div>
            )}

            {/* ===== INSPECT MODAL ===== */}
            {inspectModal && (
                <div className="fixed inset-0 bg-black/70 z-[60] flex items-center justify-center p-8" onClick={() => setInspectModal(null)}>
                    <div className="bg-surface-900 border border-surface-700/50 rounded-2xl w-full max-w-4xl max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between px-5 py-4 border-b border-surface-700/50">
                            <h3 className="text-white font-semibold flex items-center gap-2"><Info className="w-4 h-4 text-sky-400" /> Inspect: {inspectModal.name}</h3>
                            <button onClick={() => setInspectModal(null)} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white"><X className="w-4 h-4" /></button>
                        </div>
                        <pre className="flex-1 overflow-auto p-5 text-xs text-surface-200/70 font-mono whitespace-pre-wrap leading-relaxed">{inspectModal.data}</pre>
                    </div>
                </div>
            )}

            {/* ===== TEMPLATE DEPLOY MODAL ===== */}
            {templateDeploy && (
                <div className="fixed inset-0 bg-black/70 z-[60] flex items-center justify-center p-8" onClick={() => setTemplateDeploy(null)}>
                    <div className="bg-surface-900 border border-surface-700/50 rounded-2xl w-full max-w-lg" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between px-5 py-4 border-b border-surface-700/50">
                            <h3 className="text-white font-semibold flex items-center gap-2">
                                <span className="text-xl">{templateDeploy.logo}</span> Deploy {templateDeploy.title}
                            </h3>
                            <button onClick={() => setTemplateDeploy(null)} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white"><X className="w-4 h-4" /></button>
                        </div>
                        <div className="p-5 space-y-4">
                            <p className="text-sm text-surface-200/50">{templateDeploy.description}</p>
                            <p className="text-xs text-surface-200/50 font-mono">Image: {templateDeploy.image}</p>
                            <input value={templateName} onChange={e => setTemplateName(e.target.value)} placeholder="Container name"
                                className="w-full px-3 py-2 bg-surface-800/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none" />
                            {templateDeploy.env?.map(e => (
                                <div key={e.name}>
                                    <label className="text-xs text-surface-200/40 mb-1 block">{e.label}</label>
                                    <input value={templateEnv[e.name] || e.default} onChange={ev => setTemplateEnv(p => ({ ...p, [e.name]: ev.target.value }))}
                                        className="w-full px-3 py-2 bg-surface-800/50 border border-surface-700/50 rounded-lg text-white text-sm focus:border-blue-500/50 focus:outline-none font-mono" />
                                </div>
                            ))}
                            {templateDeploy.ports && <div className="flex gap-2 flex-wrap">
                                {Object.entries(templateDeploy.ports).map(([k, v]) => (
                                    <span key={k} className="text-xs px-2 py-1 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400">Port {v} → {k}</span>
                                ))}
                            </div>}
                            <button onClick={deployTemplateAction} className="w-full px-4 py-2.5 bg-gradient-to-r from-blue-600 to-blue-700 text-white rounded-lg text-sm font-medium hover:from-blue-500 hover:to-blue-600 transition-all">
                                🚀 Deploy {templateDeploy.title}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* ===== FILE BROWSER MODAL ===== */}
            {fileModal && (
                <div className="fixed inset-0 bg-black/70 z-[60] flex items-center justify-center p-8" onClick={() => setFileModal(null)}>
                    <div className="bg-surface-900 border border-surface-700/50 rounded-2xl w-full max-w-3xl max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between px-5 py-4 border-b border-surface-700/50">
                            <h3 className="text-white font-semibold flex items-center gap-2">
                                <FolderOpen className="w-4 h-4 text-indigo-400" /> Files: {fileModal.name}
                                <span className="text-xs text-surface-200/50 font-mono ml-2">{fileModal.path}</span>
                            </h3>
                            <button onClick={() => setFileModal(null)} className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white"><X className="w-4 h-4" /></button>
                        </div>
                        <div className="flex-1 overflow-auto">
                            {fileModal.path !== '/' && (
                                <button onClick={() => browseContainer(fileModal.id, fileModal.name, fileModal.path.split('/').slice(0, -1).join('/') || '/')}
                                    className="w-full text-left px-5 py-2.5 hover:bg-surface-800/50 text-blue-400 text-sm border-b border-surface-700/20 flex items-center gap-2">
                                    ← ..
                                </button>
                            )}
                            {fileModal.files.map(f => (
                                <div key={f.path} className="flex items-center gap-3 px-5 py-2.5 hover:bg-surface-800/50 border-b border-surface-700/20 cursor-pointer"
                                    onClick={() => f.is_dir && browseContainer(fileModal.id, fileModal.name, f.path)}>
                                    {f.is_dir ? <FolderOpen className="w-4 h-4 text-amber-400 shrink-0" /> : <FileText className="w-4 h-4 text-surface-200/40 shrink-0" />}
                                    <span className={`text-sm flex-1 ${f.is_dir ? 'text-amber-400 font-medium' : 'text-white'}`}>{f.name}{f.is_dir ? '/' : ''}</span>
                                    <span className="text-xs text-surface-200/50 font-mono">{f.mode}</span>
                                    {!f.is_dir && <span className="text-xs text-surface-200/50">{fmtBytes(f.size)}</span>}
                                </div>
                            ))}
                            {fileModal.files.length === 0 && <p className="text-center text-surface-200/50 py-8">Empty directory</p>}
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
