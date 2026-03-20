import { useState, useEffect } from 'react';
import { ShieldCheck, Shield, Eye, Ban, Plus, Trash2, Loader, X, AlertTriangle, CheckCircle2, XCircle, ToggleLeft, ToggleRight, Settings2, FileWarning, ListFilter, Globe } from 'lucide-react';
import { wafService } from '../services/waf';
import { useToast } from '../components/ui/ToastProvider';

interface WAFConfig {
    id: string; server_id: string; enabled: boolean; mode: string; paranoid_level: number;
    allowed_methods: string; max_request_body: number; crs_enabled: boolean;
    sqli_protection: boolean; xss_protection: boolean; rfi_protection: boolean;
    lfi_protection: boolean; rce_protection: boolean; scanner_block: boolean;
}
interface WAFRule { id: string; rule_id: number; description: string; is_disabled: boolean; created_at: string; }
interface WAFWhitelistEntry { id: string; type: string; value: string; reason: string; created_at: string; }
interface WAFLogEntry { id: number; rule_id: number; uri: string; client_ip: string; message: string; severity: string; action: string; created_at: string; }

type Tab = 'config' | 'rules' | 'whitelist' | 'logs';

const TABS: { id: Tab; label: string; icon: any }[] = [
    { id: 'config', label: 'Configuration', icon: Settings2 },
    { id: 'rules', label: 'Rules', icon: ListFilter },
    { id: 'whitelist', label: 'Whitelist', icon: CheckCircle2 },
    { id: 'logs', label: 'Audit Logs', icon: FileWarning },
];

export default function WAF() {
    const toast = useToast();
    const [tab, setTab] = useState<Tab>('config');
    const [loading, setLoading] = useState(false);
    const [serverId, setServerId] = useState('');

    const [config, setConfig] = useState<WAFConfig | null>(null);
    const [rules, setRules] = useState<WAFRule[]>([]);
    const [whitelist, setWhitelist] = useState<WAFWhitelistEntry[]>([]);
    const [logs, setLogs] = useState<WAFLogEntry[]>([]);

    const [showAddRule, setShowAddRule] = useState(false);
    const [newRuleId, setNewRuleId] = useState('');
    const [newRuleDesc, setNewRuleDesc] = useState('');

    const [showAddWl, setShowAddWl] = useState(false);
    const [newWl, setNewWl] = useState({ type: 'ip', value: '', reason: '' });

    const fetchConfig = async () => { if (!serverId) return; setLoading(true); try { const c = await wafService.getConfig(serverId); setConfig(c); } catch { toast.error('Failed to load WAF config'); } setLoading(false); };
    const fetchRules = async () => { if (!serverId) return; setLoading(true); try { const r = await wafService.listDisabledRules(serverId); setRules(r || []); } catch { toast.error('Failed'); } setLoading(false); };
    const fetchWhitelist = async () => { if (!serverId) return; setLoading(true); try { const w = await wafService.listWhitelist(serverId); setWhitelist(w || []); } catch { toast.error('Failed'); } setLoading(false); };
    const fetchLogs = async () => { if (!serverId) return; setLoading(true); try { const l = await wafService.listLogs(serverId); setLogs(l.data || []); } catch { toast.error('Failed'); } setLoading(false); };

    useEffect(() => {
        if (!serverId) return;
        if (tab === 'config') fetchConfig();
        else if (tab === 'rules') fetchRules();
        else if (tab === 'whitelist') fetchWhitelist();
        else if (tab === 'logs') fetchLogs();
    }, [tab, serverId]);

    const updateConfig = async (patch: Record<string, any>) => {
        if (!serverId) return;
        try { const c = await wafService.updateConfig(serverId, patch); setConfig(c); toast.success('WAF config updated'); } catch { toast.error('Failed to update'); }
    };

    const handleDisableRule = async () => {
        if (!serverId || !newRuleId) return;
        try { await wafService.disableRule(serverId, parseInt(newRuleId), newRuleDesc); toast.success('Rule disabled'); setShowAddRule(false); setNewRuleId(''); setNewRuleDesc(''); fetchRules(); } catch { toast.error('Failed'); }
    };
    const handleEnableRule = async (id: string) => {
        if (!confirm('Re-enable this rule?')) return;
        try { await wafService.enableRule(id); toast.success('Rule re-enabled'); fetchRules(); } catch { toast.error('Failed'); }
    };
    const handleAddWl = async () => {
        if (!serverId) return;
        try { await wafService.addWhitelist(serverId, newWl); toast.success('Whitelist entry added'); setShowAddWl(false); setNewWl({ type: 'ip', value: '', reason: '' }); fetchWhitelist(); } catch { toast.error('Failed'); }
    };
    const handleRemoveWl = async (id: string) => {
        if (!confirm('Remove this whitelist entry?')) return;
        try { await wafService.removeWhitelist(id); toast.success('Removed'); fetchWhitelist(); } catch { toast.error('Failed'); }
    };

    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-nova-500/50 text-sm";
    const btnPrimary = "px-6 py-2.5 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:from-nova-500 hover:to-nova-600 transition-all shadow-lg shadow-nova-500/20";
    const btnSecondary = "px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 hover:text-white hover:border-surface-700 transition-colors text-sm";

    const ProtectionToggle = ({ label, desc, value, field }: { label: string; desc: string; value: boolean; field: string }) => (
        <div className="flex items-center justify-between p-4 bg-surface-900/30 rounded-xl border border-surface-700/30 hover:border-surface-700/50 transition-colors">
            <div>
                <p className="text-white font-medium text-sm">{label}</p>
                <p className="text-xs text-surface-200/40 mt-0.5">{desc}</p>
            </div>
            <button onClick={() => updateConfig({ [field]: !value })} className="p-1">
                {value ? <ToggleRight className="w-8 h-8 text-emerald-400" /> : <ToggleLeft className="w-8 h-8 text-surface-200/30" />}
            </button>
        </div>
    );

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <div className="p-2.5 bg-gradient-to-br from-amber-500/20 to-red-500/10 rounded-xl border border-amber-500/20"><ShieldCheck className="w-6 h-6 text-amber-400" /></div>
                        WAF — ModSecurity + OWASP CRS
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Web Application Firewall with OWASP Core Rule Set protection</p>
                </div>
            </div>

            {/* Server Selector */}
            <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center gap-3">
                <Globe className="w-5 h-5 text-surface-200/40" />
                <input value={serverId} onChange={e => setServerId(e.target.value)} placeholder="Enter Server ID to manage WAF..."
                    className={`${inputCls} flex-1`} onKeyDown={e => e.key === 'Enter' && fetchConfig()} />
                <button onClick={fetchConfig} className={btnPrimary}>Load</button>
            </div>

            {serverId && (
                <>
                    {/* Tabs */}
                    <div className="flex gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1">
                        {TABS.map(t => (
                            <button key={t.id} onClick={() => setTab(t.id)}
                                className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all ${tab === t.id ? 'bg-amber-500/20 text-amber-400 border border-amber-500/30' : 'text-surface-200/50 hover:text-white hover:bg-surface-700/30 border border-transparent'}`}>
                                <t.icon className="w-4 h-4" /> {t.label}
                            </button>
                        ))}
                    </div>

                    {loading && <div className="flex justify-center py-12"><Loader className="w-6 h-6 animate-spin text-amber-400" /></div>}

                    {/* ════════════ CONFIG TAB ════════════ */}
                    {tab === 'config' && config && !loading && (
                        <div className="space-y-6">
                            {/* Master Toggle */}
                            <div className={`p-5 rounded-2xl border-2 ${config.enabled ? 'border-emerald-500/40 bg-emerald-500/5' : 'border-red-500/40 bg-red-500/5'}`}>
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-4">
                                        {config.enabled ? <ShieldCheck className="w-10 h-10 text-emerald-400" /> : <XCircle className="w-10 h-10 text-red-400" />}
                                        <div>
                                            <h3 className="text-lg font-bold text-white">ModSecurity WAF</h3>
                                            <p className="text-sm text-surface-200/50">{config.enabled ? 'Active — protecting your server' : 'Disabled — not filtering traffic'}</p>
                                        </div>
                                    </div>
                                    <button onClick={() => updateConfig({ enabled: !config.enabled })} className={`px-6 py-3 rounded-xl text-sm font-bold transition-all ${config.enabled ? 'bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30' : 'bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 border border-emerald-500/30'}`}>
                                        {config.enabled ? 'Disable WAF' : 'Enable WAF'}
                                    </button>
                                </div>
                            </div>

                            {/* Mode & Paranoia */}
                            <div className="grid grid-cols-2 gap-4">
                                <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4">
                                    <label className="block text-sm font-medium text-surface-200/60 mb-2">Detection Mode</label>
                                    <select value={config.mode} onChange={e => updateConfig({ mode: e.target.value })}
                                        className={inputCls + " bg-transparent"}>
                                        <option value="detection_only">Detection Only (Log but don't block)</option>
                                        <option value="blocking">Blocking (Active protection)</option>
                                    </select>
                                    <p className="text-xs text-surface-200/30 mt-1.5">
                                        {config.mode === 'blocking' ? '🛡 Malicious requests are blocked' : '👁 Threats are logged but not blocked'}
                                    </p>
                                </div>
                                <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4">
                                    <label className="block text-sm font-medium text-surface-200/60 mb-2">Paranoia Level (1-4)</label>
                                    <div className="flex items-center gap-3">
                                        {[1, 2, 3, 4].map(l => (
                                            <button key={l} onClick={() => updateConfig({ paranoid_level: l })}
                                                className={`flex-1 py-2.5 rounded-lg text-sm font-bold transition-all border ${config.paranoid_level === l ? 'bg-amber-500/20 border-amber-500/40 text-amber-400' : 'bg-surface-900/50 border-surface-700/30 text-surface-200/40 hover:text-white'}`}>
                                                PL {l}
                                            </button>
                                        ))}
                                    </div>
                                    <p className="text-xs text-surface-200/30 mt-1.5">Higher = more strict, may cause false positives</p>
                                </div>
                            </div>

                            {/* Protection Toggles */}
                            <div>
                                <h3 className="text-lg font-semibold text-white mb-3">OWASP CRS Protection Modules</h3>
                                <div className="grid grid-cols-2 gap-3">
                                    <ProtectionToggle label="SQL Injection (SQLi)" desc="Block SQL injection attacks (OWASP A03)" value={config.sqli_protection} field="sqli_protection" />
                                    <ProtectionToggle label="Cross-Site Scripting (XSS)" desc="Block XSS attacks (OWASP A07)" value={config.xss_protection} field="xss_protection" />
                                    <ProtectionToggle label="Remote File Inclusion (RFI)" desc="Block remote file include attacks" value={config.rfi_protection} field="rfi_protection" />
                                    <ProtectionToggle label="Local File Inclusion (LFI)" desc="Block directory traversal & LFI" value={config.lfi_protection} field="lfi_protection" />
                                    <ProtectionToggle label="Remote Code Execution (RCE)" desc="Block command injection attacks" value={config.rce_protection} field="rce_protection" />
                                    <ProtectionToggle label="Scanner & Bot Blocking" desc="Block vulnerability scanners & bad bots" value={config.scanner_block} field="scanner_block" />
                                    <ProtectionToggle label="OWASP CRS Engine" desc="Enable full OWASP Core Rule Set" value={config.crs_enabled} field="crs_enabled" />
                                </div>
                            </div>

                            {/* Advanced */}
                            <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-4">
                                <h3 className="text-sm font-semibold text-white mb-3">Advanced Settings</h3>
                                <div className="grid grid-cols-2 gap-4">
                                    <div>
                                        <label className="block text-xs text-surface-200/40 mb-1">Allowed HTTP Methods</label>
                                        <input value={config.allowed_methods} onChange={e => updateConfig({ allowed_methods: e.target.value })} className={inputCls} />
                                    </div>
                                    <div>
                                        <label className="block text-xs text-surface-200/40 mb-1">Max Request Body (bytes)</label>
                                        <input type="number" value={config.max_request_body} onChange={e => updateConfig({ max_request_body: parseInt(e.target.value) || 0 })} className={inputCls} />
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}

                    {/* ════════════ RULES TAB ════════════ */}
                    {tab === 'rules' && !loading && (
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <p className="text-sm text-surface-200/50">Disabled ModSecurity rules for this server</p>
                                <button onClick={() => setShowAddRule(true)} className={btnPrimary + " flex items-center gap-2"}><Plus className="w-4 h-4" /> Disable Rule</button>
                            </div>
                            <div className="space-y-2">
                                {rules.map(r => (
                                    <div key={r.id} className="group flex items-center justify-between bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-700 transition-colors">
                                        <div className="flex items-center gap-3">
                                            <div className="p-2 rounded-lg bg-red-500/10 border border-red-500/20"><Ban className="w-4 h-4 text-red-400" /></div>
                                            <div>
                                                <p className="text-white font-medium">Rule #{r.rule_id}</p>
                                                <p className="text-xs text-surface-200/40">{r.description || 'No description'}</p>
                                            </div>
                                        </div>
                                        <button onClick={() => handleEnableRule(r.id)} className="px-3 py-1.5 rounded-lg bg-emerald-500/10 text-emerald-400 text-xs font-medium border border-emerald-500/20 hover:bg-emerald-500/20 opacity-0 group-hover:opacity-100 transition-all">
                                            Re-enable
                                        </button>
                                    </div>
                                ))}
                                {rules.length === 0 && <p className="text-center text-surface-200/30 py-12">All rules are active — no rules have been disabled</p>}
                            </div>
                        </div>
                    )}

                    {/* ════════════ WHITELIST TAB ════════════ */}
                    {tab === 'whitelist' && !loading && (
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <p className="text-sm text-surface-200/50">IPs, URIs, and rules excluded from WAF filtering</p>
                                <button onClick={() => setShowAddWl(true)} className={btnPrimary + " flex items-center gap-2"}><Plus className="w-4 h-4" /> Add Exception</button>
                            </div>
                            <div className="space-y-2">
                                {whitelist.map(w => (
                                    <div key={w.id} className="group flex items-center justify-between bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 hover:border-surface-700 transition-colors">
                                        <div className="flex items-center gap-3">
                                            <div className={`p-2 rounded-lg border ${w.type === 'ip' ? 'bg-blue-500/10 border-blue-500/20' : w.type === 'uri' ? 'bg-purple-500/10 border-purple-500/20' : 'bg-amber-500/10 border-amber-500/20'}`}>
                                                <CheckCircle2 className={`w-4 h-4 ${w.type === 'ip' ? 'text-blue-400' : w.type === 'uri' ? 'text-purple-400' : 'text-amber-400'}`} />
                                            </div>
                                            <div>
                                                <div className="flex items-center gap-2">
                                                    <span className="text-xs px-2 py-0.5 rounded-full bg-surface-700/50 text-surface-200/60 uppercase font-medium">{w.type}</span>
                                                    <p className="text-white font-medium">{w.value}</p>
                                                </div>
                                                {w.reason && <p className="text-xs text-surface-200/40 mt-0.5">{w.reason}</p>}
                                            </div>
                                        </div>
                                        <button onClick={() => handleRemoveWl(w.id)} className="p-2 rounded-lg hover:bg-danger/20 text-surface-200/40 hover:text-danger transition-colors opacity-0 group-hover:opacity-100">
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </div>
                                ))}
                                {whitelist.length === 0 && <p className="text-center text-surface-200/30 py-12">No whitelist entries — all traffic is filtered</p>}
                            </div>
                        </div>
                    )}

                    {/* ════════════ LOGS TAB ════════════ */}
                    {tab === 'logs' && !loading && (
                        <div className="space-y-4">
                            <p className="text-sm text-surface-200/50">Recent WAF events and blocked requests</p>
                            <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl overflow-hidden">
                                <table className="w-full">
                                    <thead>
                                        <tr className="border-b border-surface-700/50">
                                            <th className="px-4 py-3 text-left text-xs font-medium text-surface-200/40 uppercase">Time</th>
                                            <th className="px-4 py-3 text-left text-xs font-medium text-surface-200/40 uppercase">Client IP</th>
                                            <th className="px-4 py-3 text-left text-xs font-medium text-surface-200/40 uppercase">Rule</th>
                                            <th className="px-4 py-3 text-left text-xs font-medium text-surface-200/40 uppercase">URI</th>
                                            <th className="px-4 py-3 text-left text-xs font-medium text-surface-200/40 uppercase">Severity</th>
                                            <th className="px-4 py-3 text-left text-xs font-medium text-surface-200/40 uppercase">Action</th>
                                            <th className="px-4 py-3 text-left text-xs font-medium text-surface-200/40 uppercase">Message</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-surface-700/30">
                                        {logs.map(log => (
                                            <tr key={log.id} className="hover:bg-surface-700/20 transition-colors">
                                                <td className="px-4 py-3 text-xs text-surface-200/50 whitespace-nowrap">{new Date(log.created_at).toLocaleString()}</td>
                                                <td className="px-4 py-3 text-xs text-white font-mono">{log.client_ip}</td>
                                                <td className="px-4 py-3 text-xs"><span className="px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 font-mono">{log.rule_id}</span></td>
                                                <td className="px-4 py-3 text-xs text-surface-200/60 max-w-xs truncate">{log.uri}</td>
                                                <td className="px-4 py-3 text-xs">
                                                    <span className={`px-2 py-0.5 rounded-full text-xs ${log.severity === 'critical' ? 'bg-red-500/20 text-red-400' : log.severity === 'error' ? 'bg-orange-500/20 text-orange-400' : 'bg-yellow-500/20 text-yellow-400'}`}>
                                                        {log.severity}
                                                    </span>
                                                </td>
                                                <td className="px-4 py-3 text-xs">
                                                    <span className={`px-2 py-0.5 rounded-full text-xs ${log.action === 'blocked' ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/20 text-blue-400'}`}>
                                                        {log.action}
                                                    </span>
                                                </td>
                                                <td className="px-4 py-3 text-xs text-surface-200/50 max-w-sm truncate">{log.message}</td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                                {logs.length === 0 && <p className="text-center text-surface-200/30 py-12">No WAF events recorded</p>}
                            </div>
                        </div>
                    )}
                </>
            )}

            {!serverId && (
                <div className="text-center py-16">
                    <ShieldCheck className="w-16 h-16 mx-auto mb-4 text-surface-200/10" />
                    <h3 className="text-lg font-semibold text-surface-200/40">Select a Server</h3>
                    <p className="text-sm text-surface-200/25 mt-1">Enter a Server ID above to configure its Web Application Firewall</p>
                </div>
            )}

            {/* ════════ MODALS ════════ */}

            {showAddRule && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAddRule(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-3 flex items-center gap-2"><Ban className="w-5 h-5 text-red-400" /> Disable Rule</h2>
                        <div className="space-y-3">
                            <input value={newRuleId} onChange={e => setNewRuleId(e.target.value)} placeholder="Rule ID (e.g. 942100)" className={inputCls} />
                            <input value={newRuleDesc} onChange={e => setNewRuleDesc(e.target.value)} placeholder="Reason (optional)" className={inputCls} />
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowAddRule(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleDisableRule} className={btnPrimary}>Disable</button>
                        </div>
                    </div>
                </div>
            )}

            {showAddWl && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAddWl(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-3 flex items-center gap-2"><CheckCircle2 className="w-5 h-5 text-emerald-400" /> Add Whitelist Entry</h2>
                        <div className="space-y-3">
                            <select value={newWl.type} onChange={e => setNewWl({ ...newWl, type: e.target.value })} className={inputCls + " bg-transparent"}>
                                <option value="ip">IP Address</option>
                                <option value="uri">URI Path</option>
                                <option value="rule">Rule ID</option>
                            </select>
                            <input value={newWl.value} onChange={e => setNewWl({ ...newWl, value: e.target.value })}
                                placeholder={newWl.type === 'ip' ? '192.168.1.100' : newWl.type === 'uri' ? '/api/webhook' : '942100'} className={inputCls} />
                            <input value={newWl.reason} onChange={e => setNewWl({ ...newWl, reason: e.target.value })} placeholder="Reason (optional)" className={inputCls} />
                        </div>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setShowAddWl(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleAddWl} className={btnPrimary}>Add</button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
