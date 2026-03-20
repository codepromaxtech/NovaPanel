import { useState, useEffect } from 'react';
import { Mail, Plus, Trash2, Search, Loader, X, ArrowRightLeft, AtSign, MessageSquareReply, Shield, Inbox, Globe, Send, ToggleLeft, ToggleRight, Lock, HardDrive, Copy, CheckCircle2, XCircle, ExternalLink } from 'lucide-react';
import { emailService } from '../services/emails';
import { useToast } from '../components/ui/ToastProvider';

interface EmailAccount { id: string; address: string; quota_mb: number; used_mb: number; is_active: boolean; created_at: string; domain_id: string; }
interface Forwarder { id: string; source: string; destination: string; domain_id: string; created_at: string; }
interface Alias { id: string; source: string; destination: string; domain_id: string; created_at: string; }
interface Autoresponder { id: string; account_id: string; subject: string; body: string; is_active: boolean; start_date: string | null; end_date: string | null; created_at: string; }
interface DNSRecord { found: boolean; value: string; expected: string; }
interface DNSStatus { spf: DNSRecord; dkim: DNSRecord; dmarc: DNSRecord; }

type Tab = 'accounts' | 'forwarders' | 'aliases' | 'autoresponders' | 'dns' | 'catchall' | 'webmail';

const TABS: { id: Tab; label: string; icon: any }[] = [
    { id: 'accounts', label: 'Accounts', icon: Mail },
    { id: 'forwarders', label: 'Forwarders', icon: ArrowRightLeft },
    { id: 'aliases', label: 'Aliases', icon: AtSign },
    { id: 'autoresponders', label: 'Auto-Reply', icon: MessageSquareReply },
    { id: 'dns', label: 'DNS / Auth', icon: Shield },
    { id: 'catchall', label: 'Catch-All', icon: Inbox },
    { id: 'webmail', label: 'Webmail', icon: Globe },
];

export default function Email() {
    const toast = useToast();
    const [tab, setTab] = useState<Tab>('accounts');
    const [loading, setLoading] = useState(false);
    const [search, setSearch] = useState('');

    // ──── Accounts ────
    const [accounts, setAccounts] = useState<EmailAccount[]>([]);
    const [showCreate, setShowCreate] = useState(false);
    const [newAcct, setNewAcct] = useState({ address: '', password: '', quota_mb: 1024, domain_id: '' });
    const [pwModal, setPwModal] = useState<string | null>(null);
    const [newPw, setNewPw] = useState('');
    const [quotaModal, setQuotaModal] = useState<string | null>(null);
    const [newQuota, setNewQuota] = useState(1024);

    // ──── Forwarders ────
    const [forwarders, setForwarders] = useState<Forwarder[]>([]);
    const [showFwdCreate, setShowFwdCreate] = useState(false);
    const [newFwd, setNewFwd] = useState({ source: '', destination: '', domain_id: '' });

    // ──── Aliases ────
    const [aliases, setAliases] = useState<Alias[]>([]);
    const [showAliasCreate, setShowAliasCreate] = useState(false);
    const [newAlias, setNewAlias] = useState({ source: '', destination: '', domain_id: '' });

    // ──── Autoresponders ────
    const [autoresponders, setAutoresponders] = useState<Autoresponder[]>([]);
    const [showArCreate, setShowArCreate] = useState(false);
    const [newAr, setNewAr] = useState({ account_id: '', subject: '', body: '', start_date: '', end_date: '' });

    // ──── DNS ────
    const [dnsStatus, setDnsStatus] = useState<DNSStatus | null>(null);
    const [dnsDomain, setDnsDomain] = useState('');

    // ──── Catch-All ────
    const [catchAllDomainId, setCatchAllDomainId] = useState('');
    const [catchAllAddr, setCatchAllAddr] = useState('');

    const fetchAccounts = async () => {
        setLoading(true);
        try { const r = await emailService.list(); setAccounts(r.data || []); } catch { toast.error('Failed to load accounts'); }
        setLoading(false);
    };
    const fetchForwarders = async () => {
        setLoading(true);
        try { const r = await emailService.listForwarders(); setForwarders(r.data || []); } catch { toast.error('Failed to load forwarders'); }
        setLoading(false);
    };
    const fetchAliases = async () => {
        setLoading(true);
        try { const r = await emailService.listAliases(); setAliases(r.data || []); } catch { toast.error('Failed to load aliases'); }
        setLoading(false);
    };
    const fetchAutoresponders = async () => {
        setLoading(true);
        try { const r = await emailService.listAutoresponders(); setAutoresponders(r.data || []); } catch { toast.error('Failed to load autoresponders'); }
        setLoading(false);
    };

    useEffect(() => {
        if (tab === 'accounts') fetchAccounts();
        else if (tab === 'forwarders') fetchForwarders();
        else if (tab === 'aliases') fetchAliases();
        else if (tab === 'autoresponders') fetchAutoresponders();
    }, [tab]);

    // ──── Handlers ────
    const handleCreateAcct = async () => {
        try { await emailService.create(newAcct); toast.success('Account created'); setShowCreate(false); setNewAcct({ address: '', password: '', quota_mb: 1024, domain_id: '' }); fetchAccounts(); }
        catch { toast.error('Failed to create account'); }
    };
    const handleDeleteAcct = async (id: string) => {
        if (!confirm('Delete this email account? All emails will be lost.')) return;
        try { await emailService.remove(id); toast.success('Account deleted'); fetchAccounts(); } catch { toast.error('Failed'); }
    };
    const handleToggleAcct = async (id: string, active: boolean) => {
        try { await emailService.toggleAccount(id, !active); toast.success(active ? 'Account disabled' : 'Account enabled'); fetchAccounts(); } catch { toast.error('Failed'); }
    };
    const handleChangePw = async () => {
        if (!pwModal) return;
        try { await emailService.changePassword(pwModal, newPw); toast.success('Password updated'); setPwModal(null); setNewPw(''); } catch { toast.error('Failed'); }
    };
    const handleUpdateQuota = async () => {
        if (!quotaModal) return;
        try { await emailService.updateQuota(quotaModal, newQuota); toast.success('Quota updated'); setQuotaModal(null); fetchAccounts(); } catch { toast.error('Failed'); }
    };
    const handleCreateFwd = async () => {
        try { await emailService.createForwarder(newFwd); toast.success('Forwarder created'); setShowFwdCreate(false); setNewFwd({ source: '', destination: '', domain_id: '' }); fetchForwarders(); }
        catch { toast.error('Failed'); }
    };
    const handleDeleteFwd = async (id: string) => {
        if (!confirm('Delete this forwarder?')) return;
        try { await emailService.deleteForwarder(id); toast.success('Deleted'); fetchForwarders(); } catch { toast.error('Failed'); }
    };
    const handleCreateAlias = async () => {
        try { await emailService.createAlias(newAlias); toast.success('Alias created'); setShowAliasCreate(false); setNewAlias({ source: '', destination: '', domain_id: '' }); fetchAliases(); }
        catch { toast.error('Failed'); }
    };
    const handleDeleteAlias = async (id: string) => {
        if (!confirm('Delete this alias?')) return;
        try { await emailService.deleteAlias(id); toast.success('Deleted'); fetchAliases(); } catch { toast.error('Failed'); }
    };
    const handleCreateAr = async () => {
        try { await emailService.createAutoresponder(newAr); toast.success('Autoresponder created'); setShowArCreate(false); setNewAr({ account_id: '', subject: '', body: '', start_date: '', end_date: '' }); fetchAutoresponders(); }
        catch { toast.error('Failed'); }
    };
    const handleDeleteAr = async (id: string) => {
        if (!confirm('Delete this autoresponder?')) return;
        try { await emailService.deleteAutoresponder(id); toast.success('Deleted'); fetchAutoresponders(); } catch { toast.error('Failed'); }
    };
    const handleToggleAr = async (id: string, active: boolean) => {
        try { await emailService.toggleAutoresponder(id, !active); toast.success(active ? 'Disabled' : 'Enabled'); fetchAutoresponders(); } catch { toast.error('Failed'); }
    };
    const handleCheckDNS = async () => {
        if (!dnsDomain) return;
        setLoading(true);
        try { const s = await emailService.getDNSStatus(dnsDomain); setDnsStatus(s); } catch { toast.error('Failed to check DNS'); }
        setLoading(false);
    };
    const handleSaveCatchAll = async () => {
        try { await emailService.setCatchAll(catchAllDomainId, catchAllAddr); toast.success('Catch-all updated'); } catch { toast.error('Failed'); }
    };

    const copyToClipboard = (text: string) => { navigator.clipboard.writeText(text); toast.success('Copied!'); };

    const filteredAccounts = accounts.filter(a => a.address.toLowerCase().includes(search.toLowerCase()));
    const filteredForwarders = forwarders.filter(f => f.source.toLowerCase().includes(search.toLowerCase()) || f.destination.toLowerCase().includes(search.toLowerCase()));
    const filteredAliases = aliases.filter(a => a.source.toLowerCase().includes(search.toLowerCase()) || a.destination.toLowerCase().includes(search.toLowerCase()));

    const inputCls = "w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-nova-500/50 text-sm";
    const btnPrimary = "px-6 py-2.5 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:from-nova-500 hover:to-nova-600 transition-all shadow-lg shadow-nova-500/20";
    const btnSecondary = "px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 hover:text-white hover:border-surface-700 transition-colors text-sm";

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <div className="p-2.5 bg-gradient-to-br from-nova-500/20 to-nova-600/10 rounded-xl border border-nova-500/20"><Mail className="w-6 h-6 text-nova-400" /></div>
                        Email Management
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">{accounts.length} accounts · {forwarders.length} forwarders · {aliases.length} aliases</p>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex gap-1 bg-surface-800/50 border border-surface-700/50 rounded-xl p-1 overflow-x-auto">
                {TABS.map(t => (
                    <button key={t.id} onClick={() => setTab(t.id)}
                        className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium transition-all whitespace-nowrap ${tab === t.id ? 'bg-nova-500/20 text-nova-400 border border-nova-500/30' : 'text-surface-200/50 hover:text-white hover:bg-surface-700/30 border border-transparent'}`}>
                        <t.icon className="w-4 h-4" />
                        {t.label}
                    </button>
                ))}
            </div>

            {/* Search bar + Add button */}
            {['accounts', 'forwarders', 'aliases', 'autoresponders'].includes(tab) && (
                <div className="flex gap-3">
                    <div className="relative flex-1">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-surface-200/30" />
                        <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Search..." className={`${inputCls} pl-10`} />
                    </div>
                    <button onClick={() => {
                        if (tab === 'accounts') setShowCreate(true);
                        else if (tab === 'forwarders') setShowFwdCreate(true);
                        else if (tab === 'aliases') setShowAliasCreate(true);
                        else if (tab === 'autoresponders') setShowArCreate(true);
                    }} className={btnPrimary + " flex items-center gap-2"}>
                        <Plus className="w-4 h-4" /> Add
                    </button>
                </div>
            )}

            {loading && <div className="flex justify-center py-12"><Loader className="w-6 h-6 animate-spin text-nova-400" /></div>}

            {/* ════════════ ACCOUNTS TAB ════════════ */}
            {tab === 'accounts' && !loading && (
                <div className="space-y-3">
                    {filteredAccounts.map(acct => (
                        <div key={acct.id} className="group bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center justify-between hover:border-surface-700 transition-colors">
                            <div className="flex items-center gap-4">
                                <div className={`p-2.5 rounded-xl ${acct.is_active ? 'bg-emerald-500/10 border border-emerald-500/20' : 'bg-surface-700/30 border border-surface-700/50'}`}>
                                    <Mail className={`w-5 h-5 ${acct.is_active ? 'text-emerald-400' : 'text-surface-200/30'}`} />
                                </div>
                                <div>
                                    <p className="text-white font-medium">{acct.address}</p>
                                    <div className="flex items-center gap-3 mt-0.5">
                                        <span className="text-xs text-surface-200/40"><HardDrive className="w-3 h-3 inline mr-1" />{acct.used_mb.toFixed(0)} / {acct.quota_mb} MB</span>
                                        <span className={`text-xs px-2 py-0.5 rounded-full border ${acct.is_active ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400' : 'border-red-500/30 bg-red-500/10 text-red-400'}`}>
                                            {acct.is_active ? 'Active' : 'Disabled'}
                                        </span>
                                    </div>
                                </div>
                            </div>
                            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                <button onClick={() => handleToggleAcct(acct.id, acct.is_active)} title={acct.is_active ? 'Disable' : 'Enable'}
                                    className="p-2 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white transition-colors">
                                    {acct.is_active ? <ToggleRight className="w-4 h-4 text-emerald-400" /> : <ToggleLeft className="w-4 h-4" />}
                                </button>
                                <button onClick={() => { setPwModal(acct.id); setNewPw(''); }} title="Change Password"
                                    className="p-2 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white transition-colors"><Lock className="w-4 h-4" /></button>
                                <button onClick={() => { setQuotaModal(acct.id); setNewQuota(acct.quota_mb); }} title="Update Quota"
                                    className="p-2 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white transition-colors"><HardDrive className="w-4 h-4" /></button>
                                <button onClick={() => handleDeleteAcct(acct.id)} title="Delete"
                                    className="p-2 rounded-lg hover:bg-danger/20 text-surface-200/40 hover:text-danger transition-colors"><Trash2 className="w-4 h-4" /></button>
                            </div>
                        </div>
                    ))}
                    {filteredAccounts.length === 0 && !loading && <p className="text-center text-surface-200/30 py-12">No email accounts found</p>}
                </div>
            )}

            {/* ════════════ FORWARDERS TAB ════════════ */}
            {tab === 'forwarders' && !loading && (
                <div className="space-y-3">
                    {filteredForwarders.map(fwd => (
                        <div key={fwd.id} className="group bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center justify-between hover:border-surface-700 transition-colors">
                            <div className="flex items-center gap-4">
                                <div className="p-2.5 rounded-xl bg-blue-500/10 border border-blue-500/20"><ArrowRightLeft className="w-5 h-5 text-blue-400" /></div>
                                <div>
                                    <p className="text-white font-medium">{fwd.source}</p>
                                    <p className="text-xs text-surface-200/40 flex items-center gap-1"><Send className="w-3 h-3" /> → {fwd.destination}</p>
                                </div>
                            </div>
                            <button onClick={() => handleDeleteFwd(fwd.id)} className="p-2 rounded-lg hover:bg-danger/20 text-surface-200/40 hover:text-danger transition-colors opacity-0 group-hover:opacity-100"><Trash2 className="w-4 h-4" /></button>
                        </div>
                    ))}
                    {filteredForwarders.length === 0 && <p className="text-center text-surface-200/30 py-12">No forwarders configured</p>}
                </div>
            )}

            {/* ════════════ ALIASES TAB ════════════ */}
            {tab === 'aliases' && !loading && (
                <div className="space-y-3">
                    {filteredAliases.map(alias => (
                        <div key={alias.id} className="group bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center justify-between hover:border-surface-700 transition-colors">
                            <div className="flex items-center gap-4">
                                <div className="p-2.5 rounded-xl bg-purple-500/10 border border-purple-500/20"><AtSign className="w-5 h-5 text-purple-400" /></div>
                                <div>
                                    <p className="text-white font-medium">{alias.source}</p>
                                    <p className="text-xs text-surface-200/40">→ {alias.destination}</p>
                                </div>
                            </div>
                            <button onClick={() => handleDeleteAlias(alias.id)} className="p-2 rounded-lg hover:bg-danger/20 text-surface-200/40 hover:text-danger transition-colors opacity-0 group-hover:opacity-100"><Trash2 className="w-4 h-4" /></button>
                        </div>
                    ))}
                    {filteredAliases.length === 0 && <p className="text-center text-surface-200/30 py-12">No aliases configured</p>}
                </div>
            )}

            {/* ════════════ AUTORESPONDERS TAB ════════════ */}
            {tab === 'autoresponders' && !loading && (
                <div className="space-y-3">
                    {autoresponders.map(ar => (
                        <div key={ar.id} className="group bg-surface-800/50 border border-surface-700/50 rounded-xl p-4 flex items-center justify-between hover:border-surface-700 transition-colors">
                            <div className="flex items-center gap-4">
                                <div className={`p-2.5 rounded-xl ${ar.is_active ? 'bg-amber-500/10 border border-amber-500/20' : 'bg-surface-700/30 border border-surface-700/50'}`}>
                                    <MessageSquareReply className={`w-5 h-5 ${ar.is_active ? 'text-amber-400' : 'text-surface-200/30'}`} />
                                </div>
                                <div>
                                    <p className="text-white font-medium">{ar.subject}</p>
                                    <div className="flex items-center gap-2 mt-0.5">
                                        <span className="text-xs text-surface-200/40 truncate max-w-xs">{ar.body.substring(0, 60)}...</span>
                                        {ar.start_date && <span className="text-xs text-surface-200/30">{ar.start_date} — {ar.end_date || '∞'}</span>}
                                    </div>
                                </div>
                            </div>
                            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                <button onClick={() => handleToggleAr(ar.id, ar.is_active)} title={ar.is_active ? 'Disable' : 'Enable'}
                                    className="p-2 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white transition-colors">
                                    {ar.is_active ? <ToggleRight className="w-4 h-4 text-amber-400" /> : <ToggleLeft className="w-4 h-4" />}
                                </button>
                                <button onClick={() => handleDeleteAr(ar.id)} className="p-2 rounded-lg hover:bg-danger/20 text-surface-200/40 hover:text-danger transition-colors"><Trash2 className="w-4 h-4" /></button>
                            </div>
                        </div>
                    ))}
                    {autoresponders.length === 0 && <p className="text-center text-surface-200/30 py-12">No autoresponders configured</p>}
                </div>
            )}

            {/* ════════════ DNS TAB ════════════ */}
            {tab === 'dns' && (
                <div className="space-y-4">
                    <div className="flex gap-3">
                        <input value={dnsDomain} onChange={e => setDnsDomain(e.target.value)} placeholder="Enter domain (e.g. example.com)" className={`${inputCls} flex-1`} />
                        <button onClick={handleCheckDNS} className={btnPrimary} disabled={loading}>
                            {loading ? <Loader className="w-4 h-4 animate-spin" /> : 'Check DNS'}
                        </button>
                    </div>
                    {dnsStatus && (
                        <div className="space-y-3">
                            {(['spf', 'dkim', 'dmarc'] as const).map(type => {
                                const rec = dnsStatus[type];
                                return (
                                    <div key={type} className={`bg-surface-800/50 border rounded-xl p-4 ${rec.found ? 'border-emerald-500/30' : 'border-red-500/30'}`}>
                                        <div className="flex items-center justify-between mb-2">
                                            <div className="flex items-center gap-2">
                                                {rec.found ? <CheckCircle2 className="w-5 h-5 text-emerald-400" /> : <XCircle className="w-5 h-5 text-red-400" />}
                                                <span className="text-white font-semibold uppercase">{type}</span>
                                                <span className={`text-xs px-2 py-0.5 rounded-full ${rec.found ? 'bg-emerald-500/10 text-emerald-400' : 'bg-red-500/10 text-red-400'}`}>
                                                    {rec.found ? 'Found' : 'Missing'}
                                                </span>
                                            </div>
                                        </div>
                                        {rec.found && rec.value && (
                                            <div className="bg-surface-900/50 rounded-lg p-3 flex items-start justify-between gap-2">
                                                <code className="text-xs text-emerald-300 break-all">{rec.value}</code>
                                                <button onClick={() => copyToClipboard(rec.value)} className="p-1 hover:bg-surface-700/50 rounded"><Copy className="w-3.5 h-3.5 text-surface-200/40" /></button>
                                            </div>
                                        )}
                                        {!rec.found && (
                                            <div className="bg-surface-900/50 rounded-lg p-3">
                                                <p className="text-xs text-surface-200/50 mb-1">Add this DNS TXT record:</p>
                                                <div className="flex items-start justify-between gap-2">
                                                    <code className="text-xs text-amber-300 break-all">{rec.expected}</code>
                                                    <button onClick={() => copyToClipboard(rec.expected)} className="p-1 hover:bg-surface-700/50 rounded"><Copy className="w-3.5 h-3.5 text-surface-200/40" /></button>
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    )}
                    {!dnsStatus && !loading && (
                        <div className="text-center py-12 text-surface-200/30">
                            <Shield className="w-12 h-12 mx-auto mb-3 opacity-30" />
                            <p>Enter a domain name to check SPF, DKIM, and DMARC records</p>
                        </div>
                    )}
                </div>
            )}

            {/* ════════════ CATCH-ALL TAB ════════════ */}
            {tab === 'catchall' && (
                <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-6 max-w-lg">
                    <h3 className="text-lg font-semibold text-white mb-1">Catch-All Address</h3>
                    <p className="text-sm text-surface-200/40 mb-4">All emails sent to undefined addresses on this domain will be forwarded to the catch-all address.</p>
                    <div className="space-y-3">
                        <input value={catchAllDomainId} onChange={e => setCatchAllDomainId(e.target.value)} placeholder="Domain ID" className={inputCls} />
                        <input value={catchAllAddr} onChange={e => setCatchAllAddr(e.target.value)} placeholder="catch-all@example.com (leave empty to disable)" className={inputCls} />
                        <button onClick={handleSaveCatchAll} className={btnPrimary}>Save</button>
                    </div>
                </div>
            )}

            {/* ════════════ WEBMAIL TAB ════════════ */}
            {tab === 'webmail' && (
                <div className="space-y-4">
                    <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-6">
                        <div className="flex items-center justify-between mb-4">
                            <div>
                                <h3 className="text-lg font-semibold text-white flex items-center gap-2"><Globe className="w-5 h-5 text-nova-400" /> Webmail Client</h3>
                                <p className="text-sm text-surface-200/40 mt-1">Access your email inbox directly from the browser using Roundcube webmail.</p>
                            </div>
                            <a href="/webmail" target="_blank" rel="noopener" className={btnPrimary + " flex items-center gap-2"}>
                                <ExternalLink className="w-4 h-4" /> Open in New Tab
                            </a>
                        </div>
                        <div className="bg-surface-900 rounded-xl border border-surface-700/50 overflow-hidden" style={{ height: '600px' }}>
                            <iframe src="/webmail" className="w-full h-full border-0" title="Webmail" sandbox="allow-same-origin allow-scripts allow-forms allow-popups" />
                        </div>
                    </div>
                </div>
            )}

            {/* ════════════ MODALS ════════════ */}

            {/* Create Account Modal */}
            {showCreate && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowCreate(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-md shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-xl font-bold text-white mb-4">Create Email Account</h2>
                        <div className="space-y-3">
                            <input value={newAcct.domain_id} onChange={e => setNewAcct({ ...newAcct, domain_id: e.target.value })} placeholder="Domain ID" className={inputCls} />
                            <input value={newAcct.address} onChange={e => setNewAcct({ ...newAcct, address: e.target.value })} placeholder="user@example.com" className={inputCls} />
                            <input type="password" value={newAcct.password} onChange={e => setNewAcct({ ...newAcct, password: e.target.value })} placeholder="Password (min 8 chars)" className={inputCls} />
                            <input type="number" value={newAcct.quota_mb} onChange={e => setNewAcct({ ...newAcct, quota_mb: parseInt(e.target.value) || 1024 })} placeholder="Quota (MB)" className={inputCls} />
                        </div>
                        <div className="flex justify-end gap-3 mt-5">
                            <button onClick={() => setShowCreate(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleCreateAcct} className={btnPrimary}>Create</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Change Password Modal */}
            {pwModal && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setPwModal(null)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-3 flex items-center gap-2"><Lock className="w-5 h-5 text-nova-400" /> Change Password</h2>
                        <input type="password" value={newPw} onChange={e => setNewPw(e.target.value)} placeholder="New password (min 8 chars)" className={inputCls} />
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setPwModal(null)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleChangePw} className={btnPrimary}>Update</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Update Quota Modal */}
            {quotaModal && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setQuotaModal(null)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-sm shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-lg font-bold text-white mb-3 flex items-center gap-2"><HardDrive className="w-5 h-5 text-nova-400" /> Update Quota</h2>
                        <input type="number" value={newQuota} onChange={e => setNewQuota(parseInt(e.target.value) || 0)} placeholder="Quota in MB" className={inputCls} />
                        <p className="text-xs text-surface-200/40 mt-1">{(newQuota / 1024).toFixed(1)} GB</p>
                        <div className="flex justify-end gap-3 mt-4">
                            <button onClick={() => setQuotaModal(null)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleUpdateQuota} className={btnPrimary}>Update</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Create Forwarder Modal */}
            {showFwdCreate && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowFwdCreate(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-md shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-xl font-bold text-white mb-4">Create Forwarder</h2>
                        <div className="space-y-3">
                            <input value={newFwd.domain_id} onChange={e => setNewFwd({ ...newFwd, domain_id: e.target.value })} placeholder="Domain ID" className={inputCls} />
                            <input value={newFwd.source} onChange={e => setNewFwd({ ...newFwd, source: e.target.value })} placeholder="Source: user@example.com" className={inputCls} />
                            <input value={newFwd.destination} onChange={e => setNewFwd({ ...newFwd, destination: e.target.value })} placeholder="Forward to: other@example.com" className={inputCls} />
                        </div>
                        <div className="flex justify-end gap-3 mt-5">
                            <button onClick={() => setShowFwdCreate(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleCreateFwd} className={btnPrimary}>Create</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Create Alias Modal */}
            {showAliasCreate && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAliasCreate(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-md shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-xl font-bold text-white mb-4">Create Alias</h2>
                        <div className="space-y-3">
                            <input value={newAlias.domain_id} onChange={e => setNewAlias({ ...newAlias, domain_id: e.target.value })} placeholder="Domain ID" className={inputCls} />
                            <input value={newAlias.source} onChange={e => setNewAlias({ ...newAlias, source: e.target.value })} placeholder="Alias: alias@example.com" className={inputCls} />
                            <input value={newAlias.destination} onChange={e => setNewAlias({ ...newAlias, destination: e.target.value })} placeholder="Deliver to: real@example.com" className={inputCls} />
                        </div>
                        <div className="flex justify-end gap-3 mt-5">
                            <button onClick={() => setShowAliasCreate(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleCreateAlias} className={btnPrimary}>Create</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Create Autoresponder Modal */}
            {showArCreate && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowArCreate(false)}>
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-md shadow-2xl" onClick={e => e.stopPropagation()}>
                        <h2 className="text-xl font-bold text-white mb-4">Create Autoresponder</h2>
                        <div className="space-y-3">
                            <select value={newAr.account_id} onChange={e => setNewAr({ ...newAr, account_id: e.target.value })}
                                className={inputCls + " bg-transparent"}>
                                <option value="">Select account...</option>
                                {accounts.map(a => <option key={a.id} value={a.id}>{a.address}</option>)}
                            </select>
                            <input value={newAr.subject} onChange={e => setNewAr({ ...newAr, subject: e.target.value })} placeholder="Subject: Out of Office" className={inputCls} />
                            <textarea value={newAr.body} onChange={e => setNewAr({ ...newAr, body: e.target.value })} placeholder="Reply message body..." rows={4}
                                className={inputCls + " resize-none"} />
                            <div className="grid grid-cols-2 gap-3">
                                <div>
                                    <label className="block text-xs text-surface-200/40 mb-1">Start Date</label>
                                    <input type="date" value={newAr.start_date} onChange={e => setNewAr({ ...newAr, start_date: e.target.value })} className={inputCls} />
                                </div>
                                <div>
                                    <label className="block text-xs text-surface-200/40 mb-1">End Date</label>
                                    <input type="date" value={newAr.end_date} onChange={e => setNewAr({ ...newAr, end_date: e.target.value })} className={inputCls} />
                                </div>
                            </div>
                        </div>
                        <div className="flex justify-end gap-3 mt-5">
                            <button onClick={() => setShowArCreate(false)} className={btnSecondary}>Cancel</button>
                            <button onClick={handleCreateAr} className={btnPrimary}>Create</button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
