import { useEffect, useState } from 'react';
import { useModalLock } from '../hooks/useModalLock';
import {
    Globe, Plus, Search, Filter, MoreVertical, ExternalLink,
    Shield, CheckCircle, XCircle, AlertTriangle, Trash2, Edit, Eye, Loader,
} from 'lucide-react';
import type { Domain, Server } from '../types';
import { domainService } from '../services/domains';
import { serverService } from '../services/servers';
import { phpService } from '../services/php';
import { useToast } from '../components/ui/ToastProvider';

const PHP_VERSIONS = ['8.5', '8.4', '8.3', '8.2', '8.1', '8.0', '7.4'];

export default function Domains() {
    const toast = useToast();
    const [domains, setDomains] = useState<Domain[]>([]);
    const [servers, setServers] = useState<Server[]>([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState('');
    const [showAddModal, setShowAddModal] = useState(false);
    useModalLock(showAddModal);
    const [newDomain, setNewDomain] = useState<{
        name: string;
        web_server: string;
        php_version: string;
        is_load_balancer: boolean;
        server_id: string;
        backend_server_ids: string[];
    }>({
        name: '',
        web_server: 'nginx',
        php_version: '8.2',
        is_load_balancer: false,
        server_id: '',
        backend_server_ids: []
    });

    useEffect(() => {
        loadDomains();
    }, []);

    const loadDomains = async () => {
        setLoading(true);
        try {
            const [domainsResp, serversResp] = await Promise.all([
                domainService.list(),
                serverService.list(1, 100)
            ]);
            setDomains(domainsResp.data || []);
            setServers(serversResp.data || []);
        } catch {
            // Demo data fallback handled by UI state if empty
            setDomains([]);
        } finally {
            setLoading(false);
        }
    };

    const handleAddDomain = async () => {
        try {
            await domainService.create(newDomain);
            setShowAddModal(false);
            setNewDomain({
                name: '',
                web_server: 'nginx',
                php_version: '8.2',
                is_load_balancer: false,
                server_id: '',
                backend_server_ids: []
            });
            toast.success('Domain added successfully');
            loadDomains();
        } catch (err) {
            toast.error('Failed to add domain');
            console.error('Failed to add domain', err);
        }
    };

    const handleDeleteDomain = async (id: string, name: string) => {
        if (!confirm(`Delete domain "${name}"?`)) return;
        try {
            await domainService.remove(id);
            toast.success('Domain deleted');
            loadDomains();
        } catch {
            toast.error('Failed to delete domain');
        }
    };

    const [sslLoading, setSslLoading] = useState<string | null>(null);

    const handleWildcardSSL = async (id: string) => {
        setSslLoading(id);
        try {
            await domainService.provisionWildcardSSL(id);
            toast.success('Wildcard SSL provisioning started — DNS challenge via Cloudflare');
            loadDomains();
        } catch {
            toast.error('Wildcard SSL failed — ensure Cloudflare token is configured');
        } finally { setSslLoading(null); }
    };

    const [switchingPHP, setSwitchingPHP] = useState<string | null>(null);

    const handlePHPSwitch = async (domainId: string, version: string) => {
        setSwitchingPHP(domainId);
        try {
            await phpService.switchDomain(domainId, version);
            toast.success(`PHP switched to ${version}`);
            loadDomains();
        } catch (err: any) {
            toast.error(err?.response?.data?.error || 'Failed to switch PHP version');
        } finally {
            setSwitchingPHP(null);
        }
    };

    const filteredDomains = domains.filter((d) =>
        d.name.toLowerCase().includes(search.toLowerCase())
    );

    const statusBadge = (status: string) => {
        const styles: Record<string, string> = {
            active: 'bg-success/15 text-success border-success/30',
            pending: 'bg-warning/15 text-warning border-warning/30',
            suspended: 'bg-danger/15 text-danger border-danger/30',
        };
        const icons: Record<string, any> = {
            active: CheckCircle,
            pending: AlertTriangle,
            suspended: XCircle,
        };
        const Icon = icons[status] || CheckCircle;
        return (
            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${styles[status] || styles.active}`}>
                <Icon className="w-3 h-3" />
                {status}
            </span>
        );
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white">Domains</h1>
                    <p className="text-surface-200/50 text-sm mt-1">{domains.length} domains managed</p>
                </div>
                <button
                    onClick={() => setShowAddModal(true)}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:from-nova-500 hover:to-nova-600 transition-all shadow-lg shadow-nova-500/20 hover:scale-105 active:scale-95"
                >
                    <Plus className="w-4 h-4" /> Add Domain
                </button>
            </div>

            {/* Search & Filter */}
            <div className="flex gap-3">
                <div className="relative flex-1">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-surface-200/50" />
                    <input
                        type="text"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        placeholder="Search domains..."
                        className="w-full pl-10 pr-4 py-2.5 bg-surface-800 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-nova-500/50 text-sm"
                    />
                </div>
                <button className="flex items-center gap-2 px-4 py-2.5 bg-surface-800 border border-surface-700/50 rounded-lg text-surface-200/60 hover:text-white hover:border-surface-700 transition-colors text-sm">
                    <Filter className="w-4 h-4" /> Filter
                </button>
            </div>

            {/* Domains table */}
            <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl overflow-hidden">
                <table className="w-full">
                    <thead>
                        <tr className="border-b border-surface-700/50">
                            <th className="text-left px-5 py-3.5 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Domain</th>
                            <th className="text-left px-5 py-3.5 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Type</th>
                            <th className="text-left px-5 py-3.5 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Web Server</th>
                            <th className="text-left px-5 py-3.5 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">PHP</th>
                            <th className="text-left px-5 py-3.5 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">SSL</th>
                            <th className="text-left px-5 py-3.5 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Status</th>
                            <th className="text-right px-5 py-3.5 text-xs font-semibold text-surface-200/40 uppercase tracking-wider">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {loading ? (
                            Array.from({ length: 4 }).map((_, i) => (
                                <tr key={i} className="border-b border-surface-700/30">
                                    <td className="px-5 py-4"><div className="h-4 skeleton rounded w-40" /></td>
                                    <td className="px-5 py-4"><div className="h-4 skeleton rounded w-20" /></td>
                                    <td className="px-5 py-4"><div className="h-4 skeleton rounded w-16" /></td>
                                    <td className="px-5 py-4"><div className="h-4 skeleton rounded w-14" /></td>
                                    <td className="px-5 py-4"><div className="h-4 skeleton rounded w-10" /></td>
                                    <td className="px-5 py-4"><div className="h-6 skeleton rounded-full w-20" /></td>
                                    <td className="px-5 py-4"><div className="h-4 skeleton rounded w-8 ml-auto" /></td>
                                </tr>
                            ))
                        ) : filteredDomains.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="px-5 py-16 text-center">
                                    <Globe className="w-12 h-12 text-surface-200/40 mx-auto mb-3" />
                                    <p className="text-surface-200/40 text-sm">No domains found</p>
                                </td>
                            </tr>
                        ) : (
                            filteredDomains.map((domain) => (
                                <tr key={domain.id} className="border-b border-surface-700/30 hover:bg-surface-700/20 transition-colors">
                                    <td className="px-5 py-4">
                                        <div className="flex items-center gap-3">
                                            <div className="p-2 rounded-lg bg-nova-500/10">
                                                <Globe className="w-4 h-4 text-nova-400" />
                                            </div>
                                            <div>
                                                <p className="text-sm font-medium text-white flex items-center gap-2">
                                                    {domain.name}
                                                    {domain.is_load_balancer && (
                                                        <span className="bg-nova-500/20 text-nova-400 text-[10px] uppercase font-bold px-1.5 py-0.5 rounded">
                                                            LB
                                                        </span>
                                                    )}
                                                </p>
                                                <p className="text-xs text-surface-200/50">{domain.document_root}</p>
                                            </div>
                                        </div>
                                    </td>
                                    <td className="px-5 py-4">
                                        <span className="text-sm text-surface-200/60 capitalize">{domain.type}</span>
                                    </td>
                                    <td className="px-5 py-4">
                                        <span className="text-sm text-surface-200/60 capitalize">{domain.web_server}</span>
                                    </td>
                                    <td className="px-5 py-4">
                                        {domain.php_version ? (
                                            <div className="flex items-center gap-1.5">
                                                {switchingPHP === domain.id && <Loader className="w-3 h-3 text-purple-400 animate-spin" />}
                                                <select
                                                    value={domain.php_version || '8.2'}
                                                    onChange={e => handlePHPSwitch(domain.id, e.target.value)}
                                                    disabled={switchingPHP === domain.id}
                                                    className="text-xs bg-purple-500/10 text-purple-400 border border-purple-500/20 rounded px-1.5 py-0.5 focus:outline-none cursor-pointer disabled:opacity-50"
                                                >
                                                    {PHP_VERSIONS.map(v => (
                                                        <option key={v} value={v} className="bg-surface-800 text-white">{v}</option>
                                                    ))}
                                                </select>
                                            </div>
                                        ) : (
                                            <span className="text-xs text-surface-200/40">—</span>
                                        )}
                                    </td>
                                    <td className="px-5 py-4">
                                        {domain.ssl_enabled ? (
                                            <Shield className="w-4 h-4 text-success" />
                                        ) : (
                                            <Shield className="w-4 h-4 text-surface-200/40" />
                                        )}
                                    </td>
                                    <td className="px-5 py-4">{statusBadge(domain.status)}</td>
                                    <td className="px-5 py-4">
                                        <div className="flex items-center justify-end gap-2">
                                            <button className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white transition-colors" title="Visit domain">
                                                <Eye className="w-4 h-4" />
                                            </button>
                                            <button
                                                onClick={() => handleWildcardSSL(domain.id)}
                                                disabled={sslLoading === domain.id}
                                                title="Provision Wildcard SSL"
                                                className="p-1.5 rounded-lg hover:bg-nova-500/20 text-surface-200/40 hover:text-nova-400 transition-colors disabled:opacity-40">
                                                {sslLoading === domain.id ? <Loader className="w-4 h-4 animate-spin" /> : <Shield className="w-4 h-4" />}
                                            </button>
                                            <button className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-200/40 hover:text-white transition-colors" title="Edit domain">
                                                <Edit className="w-4 h-4" />
                                            </button>
                                            <button onClick={() => handleDeleteDomain(domain.id, domain.name)}
                                                className="p-1.5 rounded-lg hover:bg-danger/20 text-surface-200/40 hover:text-danger transition-colors">
                                                <Trash2 className="w-4 h-4" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>

            {/* Add Domain Modal */}
            {showAddModal && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in">
                    <div className="bg-surface-800 border border-surface-700/50 rounded-2xl p-6 w-full max-w-lg shadow-2xl animate-fade-in">
                        <h2 className="text-xl font-bold text-white mb-6">Add New Domain</h2>

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-surface-200/60 mb-1.5">Domain Name</label>
                                <input
                                    type="text"
                                    value={newDomain.name}
                                    onChange={(e) => setNewDomain({ ...newDomain, name: e.target.value })}
                                    className="w-full px-4 py-2.5 bg-surface-900 border border-surface-700/50 rounded-lg text-white placeholder:text-surface-200/20 focus:outline-none focus:border-nova-500/50 text-sm"
                                    placeholder="example.com"
                                />
                            </div>

                            <label className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-surface-800 transition-colors ${newDomain.is_load_balancer ? 'border-nova-500/50 bg-nova-500/10' : 'border-surface-700/50 bg-surface-900/50'}`}>
                                <input
                                    type="checkbox"
                                    checked={newDomain.is_load_balancer}
                                    onChange={(e) => setNewDomain({ ...newDomain, is_load_balancer: e.target.checked })}
                                    className="w-4 h-4 rounded border-surface-600 bg-surface-800 text-nova-500 focus:ring-nova-500 focus:ring-offset-surface-900"
                                />
                                <div>
                                    <p className="text-sm font-medium text-white">Distributed Load Balancer</p>
                                    <p className="text-xs text-surface-200/50">Automatically distribute traffic across multiple worker nodes.</p>
                                </div>
                            </label>

                            {!newDomain.is_load_balancer ? (
                                <div>
                                    <label className="block text-sm font-medium text-surface-200/60 mb-1.5">Target Server</label>
                                    <select value={newDomain.server_id} onChange={e => setNewDomain({ ...newDomain, server_id: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                        <option value="">Select a server...</option>
                                        {servers.map(s => (
                                            <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
                                        ))}
                                    </select>
                                    {servers.length === 0 && (
                                        <p className="text-xs text-surface-200/40 mt-1.5">No servers available</p>
                                    )}
                                </div>
                            ) : (
                                <div>
                                    <label className="block text-sm font-medium text-surface-200/60 mb-1.5">Select Backend Nodes</label>
                                    <div className="space-y-2 max-h-48 overflow-y-auto pr-2 custom-scrollbar">
                                        {servers.map(s => (
                                            <label key={s.id} className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer hover:bg-surface-800 transition-colors ${newDomain.backend_server_ids.includes(s.id) ? 'border-nova-500/50 bg-nova-500/10' : 'border-surface-700/50 bg-surface-900/50'}`}>
                                                <input
                                                    type="checkbox"
                                                    checked={newDomain.backend_server_ids.includes(s.id)}
                                                    onChange={(e) => {
                                                        const isChecked = e.target.checked;
                                                        setNewDomain(prev => ({
                                                            ...prev,
                                                            backend_server_ids: isChecked
                                                                ? [...prev.backend_server_ids, s.id]
                                                                : prev.backend_server_ids.filter(id => id !== s.id)
                                                        }));
                                                    }}
                                                    className="w-4 h-4 rounded border-surface-600 bg-surface-800 text-nova-500 focus:ring-nova-500 focus:ring-offset-surface-900"
                                                />
                                                <div>
                                                    <p className="text-sm font-medium text-white">{s.name}</p>
                                                    <p className="text-xs text-surface-200/50">{s.ip_address}</p>
                                                </div>
                                            </label>
                                        ))}
                                        {servers.length === 0 && (
                                            <p className="text-sm text-surface-200/50 p-3 text-center border border-surface-700/50 border-dashed rounded-lg">No servers available</p>
                                        )}
                                    </div>
                                </div>
                            )}

                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-4">
                                    <div>
                                        <label className="block text-sm font-medium text-surface-200/60 mb-1.5">Web Server</label>
                                        <select value={newDomain.web_server} onChange={e => setNewDomain({ ...newDomain, web_server: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                            <option value="nginx">Nginx</option>
                                            <option value="apache">Apache</option>
                                            <option value="openlitespeed">OpenLiteSpeed</option>
                                            <option value="caddy">Caddy</option>
                                        </select>
                                    </div>
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200/60 mb-1.5">PHP Version</label>
                                    <div className="space-y-2">
                                        <select value={newDomain.php_version} onChange={e => setNewDomain({ ...newDomain, php_version: e.target.value })}
                                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                            <option value="8.5">PHP 8.5</option>
                                            <option value="8.4">PHP 8.4</option>
                                            <option value="8.3">PHP 8.3</option>
                                            <option value="8.2">PHP 8.2</option>
                                            <option value="8.1">PHP 8.1</option>
                                            <option value="8.0">PHP 8.0</option>
                                            <option value="7.4">PHP 7.4</option>
                                            <option value="">None (Static/Node/Python)</option>
                                        </select>
                                    </div>
                                </div>
                            </div>

                            <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-surface-700/50">
                                <button
                                    onClick={() => setShowAddModal(false)}
                                    className="px-4 py-2.5 rounded-lg border border-surface-700/50 text-surface-200/60 hover:text-white hover:border-surface-700 transition-colors text-sm"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={handleAddDomain}
                                    className="px-6 py-2.5 rounded-lg bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:from-nova-500 hover:to-nova-600 transition-all shadow-lg shadow-nova-500/20"
                                >
                                    Add Domain
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
