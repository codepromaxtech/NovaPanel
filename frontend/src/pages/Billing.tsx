import { useState, useEffect } from 'react';
import { CreditCard, Check, Zap, Crown, Receipt, Loader } from 'lucide-react';
import { billingService } from '../services/billing';

interface Plan {
    id: string; name: string; price_cents: number; currency: string; interval: string;
    max_domains: number; max_databases: number; max_email: number; disk_gb: number;
    is_active: boolean; featured?: boolean;
}
interface InvoiceEntry {
    id: string; plan_id: string; amount_cents: number; currency: string;
    status: string; paid_at: string; due_at: string; created_at: string;
}

export default function Billing() {
    const [plans, setPlans] = useState<Plan[]>([]);
    const [invoices, setInvoices] = useState<InvoiceEntry[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        Promise.all([billingService.listPlans(), billingService.listInvoices()])
            .then(([p, i]) => {
                setPlans(Array.isArray(p) ? p : []);
                setInvoices(Array.isArray(i) ? i : []);
            })
            .catch(() => { setPlans([]); setInvoices([]); })
            .finally(() => setLoading(false));
    }, []);

    if (loading) return (
        <div className="flex items-center justify-center py-20 animate-fade-in">
            <Loader className="w-6 h-6 text-nova-500 animate-spin" />
        </div>
    );

    return (
        <div className="space-y-8 animate-fade-in">
            <div>
                <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                    <CreditCard className="w-7 h-7 text-nova-400" />
                    Billing & Plans
                </h1>
                <p className="text-surface-200/50 text-sm mt-1">Manage your subscription and view invoices</p>
            </div>

            {plans.length === 0 ? (
                <div className="glass-card rounded-2xl flex flex-col items-center justify-center py-16 text-surface-200/40">
                    <CreditCard className="w-10 h-10 mb-3 opacity-40" />
                    <p className="text-sm">No billing plans configured</p>
                    <p className="text-xs mt-1">Contact your administrator to set up billing</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {plans.map((plan, idx) => {
                        const isFeatured = idx === 1;
                        return (
                            <div key={plan.id} className={`glass-card rounded-2xl p-6 relative overflow-hidden transition-all hover:border-white/10 ${isFeatured ? 'border-nova-500/30 ring-1 ring-nova-500/20' : ''}`}>
                                {isFeatured && (
                                    <div className="absolute top-0 right-0 px-3 py-1 rounded-bl-xl bg-gradient-to-r from-nova-500 to-nova-600 text-xs font-bold text-white flex items-center gap-1">
                                        <Zap className="w-3 h-3" /> POPULAR
                                    </div>
                                )}
                                <div className="mb-4">
                                    <h3 className="text-lg font-bold text-white flex items-center gap-2">
                                        {plan.name === 'Enterprise' && <Crown className="w-4 h-4 text-warning" />}
                                        {plan.name}
                                    </h3>
                                    <div className="mt-2">
                                        <span className="text-3xl font-bold text-white">${(plan.price_cents / 100).toFixed(2)}</span>
                                        <span className="text-surface-200/40 text-sm">/{plan.interval}</span>
                                    </div>
                                </div>
                                <ul className="space-y-2.5 mb-6">
                                    <li className="flex items-center gap-2 text-sm text-surface-200/60"><Check className="w-4 h-4 text-success" />{plan.max_domains} Domains</li>
                                    <li className="flex items-center gap-2 text-sm text-surface-200/60"><Check className="w-4 h-4 text-success" />{plan.max_databases} Databases</li>
                                    <li className="flex items-center gap-2 text-sm text-surface-200/60"><Check className="w-4 h-4 text-success" />{plan.max_email} Email Accounts</li>
                                    <li className="flex items-center gap-2 text-sm text-surface-200/60"><Check className="w-4 h-4 text-success" />{plan.disk_gb} GB Storage</li>
                                </ul>
                                <button className={`w-full py-2.5 rounded-xl text-sm font-medium transition-all ${isFeatured
                                        ? 'bg-gradient-to-r from-nova-600 to-nova-700 text-white hover:shadow-lg hover:shadow-nova-500/25'
                                        : 'border border-white/10 text-surface-200/60 hover:bg-white/5'
                                    }`}>
                                    Select Plan
                                </button>
                            </div>
                        );
                    })}
                </div>
            )}

            <div className="glass-card rounded-2xl overflow-hidden">
                <div className="px-5 py-3 border-b border-white/5 flex items-center gap-2">
                    <Receipt className="w-4 h-4 text-surface-200/40" />
                    <h3 className="text-sm font-semibold text-white">Invoice History</h3>
                </div>
                {invoices.length === 0 ? (
                    <div className="flex items-center justify-center py-12 text-surface-200/40 text-sm">No invoices yet</div>
                ) : (
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-white/5">
                                <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Invoice</th>
                                <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Amount</th>
                                <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Status</th>
                                <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Date</th>
                            </tr>
                        </thead>
                        <tbody>
                            {invoices.map(inv => (
                                <tr key={inv.id} className="border-b border-white/5 hover:bg-white/[0.02] transition-colors">
                                    <td className="py-2.5 px-5 text-sm text-white font-mono">{inv.id.substring(0, 8)}</td>
                                    <td className="py-2.5 px-5 text-sm text-white font-medium">${(inv.amount_cents / 100).toFixed(2)}</td>
                                    <td className="py-2.5 px-5">
                                        <span className="text-xs px-2 py-0.5 rounded-full bg-success/10 text-success border border-success/20">{inv.status}</span>
                                    </td>
                                    <td className="py-2.5 px-5 text-sm text-surface-200/40">{inv.created_at ? new Date(inv.created_at).toLocaleDateString() : '—'}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
}
