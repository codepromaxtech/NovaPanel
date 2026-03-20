import { useState, useEffect } from 'react';
import { Cloud, Globe, Shield, Lock, Trash2, Plus, RefreshCw, Loader, X, Zap, Eye, BarChart3, Settings2, Network, Play, Square, Download } from 'lucide-react';
import { cloudflareService } from '../services/cloudflare';
import { serverService } from '../services/servers';
import { useToast } from '../components/ui/ToastProvider';

type Tab = 'zones' | 'dns' | 'ssl' | 'cache' | 'security' | 'analytics' | 'settings' | 'tunnels';
interface CFAuth { api_key: string; email?: string; }
const TABS: { id: Tab; label: string; icon: any }[] = [
    { id: 'zones', label: 'Zones', icon: Globe },
    { id: 'dns', label: 'DNS', icon: Globe },
    { id: 'ssl', label: 'SSL/TLS', icon: Lock },
    { id: 'cache', label: 'Cache', icon: Zap },
    { id: 'security', label: 'Security', icon: Shield },
    { id: 'tunnels', label: 'Tunnels', icon: Network },
    { id: 'analytics', label: 'Analytics', icon: BarChart3 },
    { id: 'settings', label: 'Settings', icon: Settings2 },
];

export default function Cloudflare() {
    const toast = useToast();
    const [tab, setTab] = useState<Tab>('zones');
    const [loading, setLoading] = useState(false);
    const [connected, setConnected] = useState(false);
    const [apiKey, setApiKey] = useState(() => localStorage.getItem('cf_api_key') || '');
    const [email, setEmail] = useState(() => localStorage.getItem('cf_email') || '');
    const auth: CFAuth = { api_key: apiKey, email: email || undefined };

    // Zones
    const [zones, setZones] = useState<any[]>([]);
    const [selectedZone, setSelectedZone] = useState('');
    const [selectedZoneName, setSelectedZoneName] = useState('');

    // DNS
    const [dnsRecords, setDnsRecords] = useState<any[]>([]);
    const [showAddDNS, setShowAddDNS] = useState(false);
    const [dnsForm, setDnsForm] = useState({ type: 'A', name: '', content: '', ttl: 1, proxied: true });

    // SSL
    const [sslMode, setSSLMode] = useState('');

    // Security
    const [securityLevel, setSecurityLevel] = useState('');

    // Analytics
    const [analyticsData, setAnalyticsData] = useState<any>(null);

    // Settings
    const [settings, setSettings] = useState<any>(null);

    // Cache purge
    const [purgeUrls, setPurgeUrls] = useState('');

    // Tunnels
    const [tunnels, setTunnels] = useState<any[]>([]);
    const [showCreateTunnel, setShowCreateTunnel] = useState(false);
    const [tunnelForm, setTunnelForm] = useState({ name: '', secret: '' });
    const [showDeployTunnel, setShowDeployTunnel] = useState<any>(null);
    const [servers, setServers] = useState<any[]>([]);
    const [deployServer, setDeployServer] = useState('');
    const [tunnelOutput, setTunnelOutput] = useState('');

    const handleConnect = async () => {
        if (!apiKey) { toast.error('API key required'); return; }
        setLoading(true);
        try {
            const r = await cloudflareService.verify(auth);
            if (r.success) {
                setConnected(true);
                localStorage.setItem('cf_api_key', apiKey);
                if (email) localStorage.setItem('cf_email', email);
                toast.success('Connected to Cloudflare!');
                fetchZones();
            } else { toast.error('Invalid credentials'); }
        } catch { toast.error('Connection failed'); }
        setLoading(false);
    };
    const handleDisconnect = () => { setConnected(false); setZones([]); setSelectedZone(''); localStorage.removeItem('cf_api_key'); localStorage.removeItem('cf_email'); };

    const fetchZones = async () => { setLoading(true); try { const r = await cloudflareService.listZones(auth); setZones(r.result || []); } catch { } setLoading(false); };
    const fetchDNS = async () => { if (!selectedZone) return; setLoading(true); try { const r = await cloudflareService.listDNS(auth, selectedZone); setDnsRecords(r.result || []); } catch { } setLoading(false); };
    const fetchSSL = async () => { if (!selectedZone) return; try { const r = await cloudflareService.getSSL(auth, selectedZone); setSSLMode(r.result?.value || ''); } catch { } };
    const fetchAnalytics = async () => { if (!selectedZone) return; setLoading(true); try { const r = await cloudflareService.analytics(auth, selectedZone); setAnalyticsData(r.result || null); } catch { } setLoading(false); };
    const fetchSettings = async () => { if (!selectedZone) return; setLoading(true); try { const r = await cloudflareService.getSettings(auth, selectedZone); setSettings(r); } catch { } setLoading(false); };
    const fetchTunnels = async () => { setLoading(true); try { const r = await cloudflareService.listTunnels(auth); setTunnels(r.result || []); } catch { } setLoading(false); };

    useEffect(() => { if (apiKey && localStorage.getItem('cf_api_key')) handleConnect(); }, []);
    useEffect(() => { if (connected && selectedZone) { fetchDNS(); fetchSSL(); } }, [selectedZone]);
    useEffect(() => { serverService.list(1, 100).then(r => setServers(r.data || [])).catch(() => { }); }, []);

    const selectZone = (z: any) => { setSelectedZone(z.id); setSelectedZoneName(z.name); setTab('dns'); };

    // DNS actions
    const handleAddDNS = async () => {
        if (!dnsForm.name || !dnsForm.content) { toast.error('Fill all fields'); return; }
        try { await cloudflareService.createDNS(auth, selectedZone, dnsForm); toast.success('Record created'); setShowAddDNS(false); fetchDNS(); } catch { toast.error('Failed'); }
    };
    const handleDeleteDNS = async (id: string) => { if (!confirm('Delete?')) return; try { await cloudflareService.deleteDNS(auth, selectedZone, id); toast.success('Deleted'); fetchDNS(); } catch { toast.error('Failed'); } };

    const handleSetSSL = async (mode: string) => { try { await cloudflareService.setSSL(auth, selectedZone, mode); setSSLMode(mode); toast.success(`SSL set to ${mode}`); } catch { toast.error('Failed'); } };
    const handlePurgeAll = async () => { if (!confirm('Purge entire cache?')) return; try { await cloudflareService.purgeAll(auth, selectedZone); toast.success('Cache purged!'); } catch { toast.error('Failed'); } };
    const handlePurgeUrls = async () => { const urls = purgeUrls.split('\n').filter(u => u.trim()); if (!urls.length) return; try { await cloudflareService.purgeURLs(auth, selectedZone, urls); toast.success('URLs purged'); setPurgeUrls(''); } catch { toast.error('Failed'); } };
    const handleSetSecurity = async (level: string) => { try { await cloudflareService.setSecurity(auth, selectedZone, level); setSecurityLevel(level); toast.success(`Security: ${level}`); } catch { toast.error('Failed'); } };
    const handleUpdateSetting = async (setting: string, value: string) => { try { await cloudflareService.updateSetting(auth, selectedZone, setting, value); toast.success('Updated'); fetchSettings(); } catch { toast.error('Failed'); } };

    // Tunnel actions
    const handleCreateTunnel = async () => {
        if (!tunnelForm.name) { toast.error('Name required'); return; }
        const secret = tunnelForm.secret || btoa(crypto.getRandomValues(new Uint8Array(32)).reduce((a, b) => a + String.fromCharCode(b), ''));
        try { await cloudflareService.createTunnel(auth, tunnelForm.name, secret); toast.success('Tunnel created'); setShowCreateTunnel(false); setTunnelForm({ name: '', secret: '' }); fetchTunnels(); } catch { toast.error('Failed'); }
    };
    const handleDeleteTunnel = async (id: string) => { if (!confirm('Delete this tunnel?')) return; try { await cloudflareService.deleteTunnel(auth, id); toast.success('Deleted'); fetchTunnels(); } catch { toast.error('Failed'); } };
    const handleDeployTunnel = async (tunnel: any) => {
        if (!deployServer) { toast.error('Select server'); return; }
        setTunnelOutput('Getting token and deploying...\n');
        try {
            const tokenR = await cloudflareService.getTunnelToken(auth, tunnel.id);
            const token = tokenR.result;
            if (!token) { toast.error('Failed to get tunnel token'); return; }
            const r = await cloudflareService.runTunnel(deployServer, token, tunnel.name);
            setTunnelOutput(r.output || 'Deployed');
            toast.success('Tunnel deployed on server!');
        } catch { toast.error('Deploy failed'); }
    };
    const handleStopTunnel = async (tunnelName: string) => {
        if (!deployServer) { toast.error('Select server'); return; }
        try { const r = await cloudflareService.stopTunnel(deployServer, tunnelName); setTunnelOutput(r.output || 'Stopped'); toast.success('Stopped'); } catch { toast.error('Failed'); }
    };
    const handleInstallCloudflared = async () => {
        if (!deployServer) { toast.error('Select server'); return; }
        setTunnelOutput('Installing cloudflared...\n');
        try { const r = await cloudflareService.installCloudflared(deployServer); setTunnelOutput(r.output || 'Installed'); toast.success('cloudflared installed'); } catch { toast.error('Failed'); }
    };

    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-orange-500/50 text-sm";
    const btnPrimary = "px-6 py-2.5 rounded-lg bg-gradient-to-r from-orange-600 to-orange-700 text-white text-sm font-medium hover:from-orange-500 hover:to-orange-600 transition-all shadow-lg shadow-orange-500/20";

    // ──── NOT CONNECTED ────
    if (!connected) {
        return (
            <div className="space-y-6 max-w-md mx-auto mt-12">
                <div className="text-center">
                    <div className="inline-flex p-4 bg-gradient-to-br from-orange-500/20 to-amber-500/10 rounded-2xl border border-orange-500/20 mb-4"><Cloud className="w-10 h-10 text-orange-400" /></div>
                    <h1 className="text-2xl font-bold text-white">Connect Cloudflare</h1>
                    <p className="text-surface-200/50 text-sm mt-2">Enter your API Token or Global API Key + Email</p>
                </div>
                <div className="space-y-3 bg-surface-800/50 border border-surface-700/50 rounded-2xl p-6">
                    <input value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="API Token or Global API Key" type="password" className={inputCls} />
                    <input value={email} onChange={e => setEmail(e.target.value)} placeholder="Email (only for Global API Key)" className={inputCls} />
                    <p className="text-xs text-surface-200/30">Leave email empty if using API Token</p>
                    <button onClick={handleConnect} disabled={loading} className={btnPrimary + " w-full flex items-center justify-center gap-2"}>
                        {loading ? <Loader className="w-4 h-4 animate-spin" /> : <Cloud className="w-4 h-4" />} Connect
                    </button>
                </div>
            </div>
        );
    }

    // ──── CONNECTED ────
    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <div className="p-2.5 bg-gradient-to-br from-orange-500/20 to-amber-500/10 rounded-xl border border-orange-500/20"><Cloud className="w-6 h-6 text-orange-400" /></div>
                        Cloudflare {selectedZoneName && <span className="text-surface-200/40 text-lg">— {selectedZoneName}</span>}
                    </h1>
                </div>
                <div className="flex items-center gap-2">
                    <span className="px-2 py-1 rounded-full bg-emerald-500/20 text-emerald-400 text-xs">Connected</span>
                    <button onClick={handleDisconnect} className="text-xs text-surface-200/40 hover:text-red-400">Disconnect</button>
                </div>
            </div>

            {selectedZone && (<div className="flex gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1 flex-wrap">
                <button onClick={() => { setSelectedZone(''); setTab('zones'); }} className="px-3 py-2 rounded-lg text-xs text-surface-200/50 hover:text-white">← All Zones</button>
                {TABS.filter(t => t.id !== 'zones').map(t => (
                    <button key={t.id} onClick={() => { setTab(t.id); if (t.id === 'analytics') fetchAnalytics(); if (t.id === 'settings') fetchSettings(); if (t.id === 'tunnels') fetchTunnels(); }}
                        className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium transition-all ${tab === t.id ? 'bg-orange-500/20 text-orange-400 border border-orange-500/30' : 'text-surface-200/50 hover:text-white border border-transparent'}`}>
                        <t.icon className="w-3.5 h-3.5" /> {t.label}
                    </button>
                ))}
            </div>)}
            {!selectedZone && connected && (
                <div className="flex gap-2"><button onClick={() => { setTab('tunnels'); fetchTunnels(); }} className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium ${tab === 'tunnels' ? 'bg-orange-500/20 text-orange-400 border border-orange-500/30' : 'bg-surface-800/50 text-surface-200/50 border border-surface-700/50'}`}><Network className="w-4 h-4" /> Tunnels</button></div>
            )}

            {loading && <div className="flex justify-center py-8"><Loader className="w-6 h-6 animate-spin text-orange-400" /></div>}

            {/* ════ ZONES ════ */}
            {(tab === 'zones' || !selectedZone) && !loading && (
                <div className="grid grid-cols-2 gap-3">{zones.map(z => (
                    <button key={z.id} onClick={() => selectZone(z)} className="text-left bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-orange-500/30 transition-colors">
                        <div className="flex items-center justify-between">
                            <p className="text-white font-semibold">{z.name}</p>
                            <span className={`px-2 py-0.5 text-xs rounded-full ${z.status === 'active' ? 'bg-emerald-500/20 text-emerald-400' : 'bg-amber-500/20 text-amber-400'}`}>{z.status}</span>
                        </div>
                        <p className="text-xs text-surface-200/40 mt-1">{z.plan?.name || 'Free'} · NS: {z.name_servers?.slice(0, 2).join(', ')}</p>
                    </button>
                ))}{zones.length === 0 && <p className="col-span-2 text-center text-surface-200/30 py-8">No zones found</p>}</div>
            )}

            {/* ════ DNS ════ */}
            {tab === 'dns' && selectedZone && !loading && (
                <div className="space-y-3">
                    <div className="flex justify-end"><button onClick={() => setShowAddDNS(true)} className={btnPrimary + " flex items-center gap-2 text-xs !px-4 !py-2"}><Plus className="w-3.5 h-3.5" /> Add Record</button></div>
                    <div className="space-y-1">{dnsRecords.map(r => (
                        <div key={r.id} className="group flex items-center justify-between bg-surface-800/50 border border-surface-700/50 rounded-lg px-4 py-2.5 hover:border-surface-700">
                            <div className="flex items-center gap-3 flex-1 min-w-0">
                                <span className="px-2 py-0.5 rounded bg-surface-700/50 text-xs font-mono text-orange-400 w-14 text-center">{r.type}</span>
                                <span className="text-white text-sm font-medium truncate">{r.name}</span>
                                <span className="text-surface-200/40 text-xs truncate">{r.content}</span>
                                {r.proxied && <Cloud className="w-3.5 h-3.5 text-orange-400 shrink-0" />}
                                <span className="text-surface-200/20 text-xs">{r.ttl === 1 ? 'Auto' : `${r.ttl}s`}</span>
                            </div>
                            <button onClick={() => handleDeleteDNS(r.id)} className="p-1.5 rounded hover:bg-red-500/20 text-surface-200/30 hover:text-red-400 opacity-0 group-hover:opacity-100"><Trash2 className="w-3.5 h-3.5" /></button>
                        </div>
                    ))}{dnsRecords.length === 0 && <p className="text-center text-surface-200/30 py-8">No DNS records</p>}</div>
                </div>
            )}

            {/* ════ SSL ════ */}
            {tab === 'ssl' && selectedZone && (
                <div className="space-y-4 max-w-lg">
                    <h2 className="text-lg font-bold text-white">SSL/TLS Mode</h2>
                    <div className="grid grid-cols-2 gap-3">
                        {['off', 'flexible', 'full', 'strict'].map(m => (
                            <button key={m} onClick={() => handleSetSSL(m)} className={`p-4 rounded-xl border text-left transition-all ${sslMode === m ? 'bg-orange-500/20 border-orange-500/30 text-orange-400' : 'bg-surface-800/50 border-surface-700/50 text-surface-200/60 hover:border-surface-700'}`}>
                                <p className="font-semibold capitalize">{m === 'strict' ? 'Full (Strict)' : m}</p>
                                <p className="text-xs mt-1 opacity-60">{m === 'off' ? 'No encryption' : m === 'flexible' ? 'CF → user encrypted' : m === 'full' ? 'End-to-end (self-signed OK)' : 'End-to-end (valid cert)'}</p>
                            </button>
                        ))}
                    </div>
                </div>
            )}

            {/* ════ CACHE ════ */}
            {tab === 'cache' && selectedZone && (
                <div className="space-y-4 max-w-lg">
                    <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-5">
                        <h3 className="text-white font-semibold mb-3">Purge Cache</h3>
                        <div className="flex gap-3 mb-3"><button onClick={handlePurgeAll} className="px-4 py-2 rounded-lg bg-red-500/20 text-red-400 text-xs border border-red-500/30 hover:bg-red-500/30">Purge Everything</button></div>
                        <textarea value={purgeUrls} onChange={e => setPurgeUrls(e.target.value)} placeholder="Specific URLs (one per line)" rows={3} className={inputCls + " font-mono"} />
                        <button onClick={handlePurgeUrls} className="mt-2 px-4 py-2 rounded-lg bg-orange-500/20 text-orange-400 text-xs border border-orange-500/30">Purge URLs</button>
                    </div>
                </div>
            )}

            {/* ════ SECURITY ════ */}
            {tab === 'security' && selectedZone && (
                <div className="space-y-4 max-w-lg">
                    <h2 className="text-lg font-bold text-white">Security Level</h2>
                    <div className="grid grid-cols-3 gap-2">
                        {[{ v: 'essentially_off', l: 'Off' }, { v: 'low', l: 'Low' }, { v: 'medium', l: 'Medium' }, { v: 'high', l: 'High' }, { v: 'under_attack', l: '🛡️ Under Attack' }].map(s => (
                            <button key={s.v} onClick={() => handleSetSecurity(s.v)} className={`p-3 rounded-xl border text-sm font-medium transition-all ${securityLevel === s.v ? 'bg-orange-500/20 border-orange-500/30 text-orange-400' : 'bg-surface-800/50 border-surface-700/50 text-surface-200/60 hover:border-surface-700'}`}>
                                {s.l}
                            </button>
                        ))}
                    </div>
                </div>
            )}

            {/* ════ ANALYTICS ════ */}
            {tab === 'analytics' && selectedZone && !loading && (
                <div className="space-y-4">
                    <div className="flex justify-end"><button onClick={fetchAnalytics} className="p-2 rounded-lg bg-surface-700/50 hover:bg-surface-700 text-white"><RefreshCw className="w-4 h-4" /></button></div>
                    {analyticsData ? (
                        <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-96 overflow-auto border border-surface-700/30">{JSON.stringify(analyticsData, null, 2)}</pre>
                    ) : (
                        <p className="text-center text-surface-200/30 py-8">No analytics data</p>
                    )}
                </div>
            )}

            {/* ════ SETTINGS ════ */}
            {tab === 'settings' && selectedZone && !loading && (
                <div className="space-y-3 max-w-lg">
                    {[
                        { key: 'always_use_https', label: 'Always Use HTTPS', values: ['on', 'off'] },
                        { key: 'rocket_loader', label: 'Rocket Loader', values: ['on', 'off'] },
                        { key: 'development_mode', label: 'Development Mode', values: ['on', 'off'] },
                    ].map(s => {
                        const current = settings?.[s.key === 'always_use_https' ? 'always_https' : s.key]?.result?.value;
                        return (
                            <div key={s.key} className="flex items-center justify-between bg-surface-800/50 border border-surface-700/50 rounded-xl p-4">
                                <div><p className="text-white font-medium text-sm">{s.label}</p><p className="text-xs text-surface-200/40">{current || 'unknown'}</p></div>
                                <div className="flex gap-2">{s.values.map(v => (
                                    <button key={v} onClick={() => handleUpdateSetting(s.key, v)} className={`px-3 py-1.5 rounded-lg text-xs border transition-all ${current === v ? 'bg-orange-500/20 text-orange-400 border-orange-500/30' : 'bg-surface-700/30 text-surface-200/50 border-surface-700/50'}`}>{v}</button>
                                ))}</div>
                            </div>
                        );
                    })}
                </div>
            )}

            {/* ════ ADD DNS MODAL ════ */}
            {showAddDNS && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAddDNS(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4">Add DNS Record</h2>
                        <div className="space-y-3">
                            <select value={dnsForm.type} onChange={e => setDnsForm({ ...dnsForm, type: e.target.value })} className={inputCls + " bg-transparent"}>
                                {['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA'].map(t => <option key={t} value={t}>{t}</option>)}
                            </select>
                            <input value={dnsForm.name} onChange={e => setDnsForm({ ...dnsForm, name: e.target.value })} placeholder="Name (@ for root)" className={inputCls} />
                            <input value={dnsForm.content} onChange={e => setDnsForm({ ...dnsForm, content: e.target.value })} placeholder="Content (IP or value)" className={inputCls} />
                            <div className="flex items-center gap-3">
                                <label className="flex items-center gap-2 text-sm text-surface-200/60"><input type="checkbox" checked={dnsForm.proxied} onChange={e => setDnsForm({ ...dnsForm, proxied: e.target.checked })} className="rounded" /> Proxied (orange cloud)</label>
                            </div>
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowAddDNS(false)} className="px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 text-sm">Cancel</button>
                            <button onClick={handleAddDNS} className={btnPrimary}>Add Record</button>
                        </div>
                    </div>
                </div>
            )}

            {/* ════ TUNNELS ════ */}
            {tab === 'tunnels' && !loading && (
                <div className="space-y-4">
                    <div className="flex items-center justify-between">
                        <h2 className="text-lg font-bold text-white flex items-center gap-2"><Network className="w-5 h-5 text-orange-400" /> Cloudflare Tunnels</h2>
                        <div className="flex gap-2">
                            <button onClick={fetchTunnels} className="p-2 rounded-lg bg-surface-700/50 hover:bg-surface-700 text-white"><RefreshCw className="w-4 h-4" /></button>
                            <button onClick={() => setShowCreateTunnel(true)} className={btnPrimary + " flex items-center gap-2 text-xs !px-4 !py-2"}><Plus className="w-3.5 h-3.5" /> New Tunnel</button>
                        </div>
                    </div>
                    <div className="flex gap-3 items-end">
                        <div className="flex-1"><label className="block text-xs text-surface-200/40 mb-1">Deploy Target Server</label><select value={deployServer} onChange={e => setDeployServer(e.target.value)} className={inputCls + " bg-transparent"}><option value="">Select server</option>{servers.map(s => <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>)}</select></div>
                        <button onClick={handleInstallCloudflared} disabled={!deployServer} className="px-3 py-2.5 rounded-lg bg-surface-700/50 text-surface-200/60 text-xs hover:bg-surface-700"><Download className="w-4 h-4 inline mr-1" />Install cloudflared</button>
                    </div>
                    <div className="space-y-2">{tunnels.map((t: any) => (
                        <div key={t.id} className="group bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-700 transition-colors">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                    <Network className="w-4 h-4 text-orange-400" />
                                    <div>
                                        <p className="text-white font-semibold text-sm">{t.name}</p>
                                        <p className="text-xs text-surface-200/30">{t.id}</p>
                                    </div>
                                    <span className={`px-2 py-0.5 text-xs rounded-full ${t.status === 'healthy' || t.status === 'active' ? 'bg-emerald-500/20 text-emerald-400' : t.status === 'inactive' ? 'bg-surface-700/50 text-surface-200/40' : 'bg-amber-500/20 text-amber-400'}`}>{t.status || 'created'}</span>
                                </div>
                                <div className="flex items-center gap-2">
                                    <button onClick={() => handleDeployTunnel(t)} disabled={!deployServer} className="px-3 py-1.5 rounded-lg bg-emerald-500/20 text-emerald-400 text-xs border border-emerald-500/30 hover:bg-emerald-500/30" title="Deploy to server"><Play className="w-3 h-3 inline mr-1" />Deploy</button>
                                    <button onClick={() => handleStopTunnel(t.name)} disabled={!deployServer} className="px-3 py-1.5 rounded-lg bg-red-500/20 text-red-400 text-xs border border-red-500/30" title="Stop on server"><Square className="w-3 h-3 inline mr-1" />Stop</button>
                                    <button onClick={() => handleDeleteTunnel(t.id)} className="p-1.5 rounded hover:bg-red-500/20 text-surface-200/30 hover:text-red-400 opacity-0 group-hover:opacity-100"><Trash2 className="w-3.5 h-3.5" /></button>
                                </div>
                            </div>
                            {t.connections && t.connections.length > 0 && <p className="text-xs text-surface-200/40 mt-2">{t.connections.length} active connection(s)</p>}
                        </div>
                    ))}{tunnels.length === 0 && <p className="text-center text-surface-200/30 py-8">No tunnels found. Create one to get started.</p>}</div>
                    {tunnelOutput && <pre className="bg-surface-900 rounded-xl p-4 text-xs text-surface-200/60 whitespace-pre-wrap font-mono max-h-48 overflow-auto border border-surface-700/30">{tunnelOutput}</pre>}
                </div>
            )}

            {/* ════ CREATE TUNNEL MODAL ════ */}
            {showCreateTunnel && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowCreateTunnel(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-4 flex items-center gap-2"><Network className="w-5 h-5 text-orange-400" /> Create Tunnel</h2>
                        <div className="space-y-3">
                            <input value={tunnelForm.name} onChange={e => setTunnelForm({ ...tunnelForm, name: e.target.value })} placeholder="Tunnel name" className={inputCls} />
                            <p className="text-xs text-surface-200/30">A random secret will be generated automatically</p>
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowCreateTunnel(false)} className="px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 text-sm">Cancel</button>
                            <button onClick={handleCreateTunnel} className={btnPrimary}>Create</button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
