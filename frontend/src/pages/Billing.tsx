import { useState, useEffect } from 'react';
import { CreditCard, Check, Zap, Crown, Receipt, Loader, X, ExternalLink } from 'lucide-react';
import { billingService } from '../services/billing';
import { useToast } from '../components/ui/ToastProvider';

interface Plan {
    id: string; name: string; price_cents: number; currency: string; interval: string;
    plan_type: string; max_domains: number; max_databases: number; max_email: number;
    disk_gb: number; max_servers: number; allow_waf: boolean; allow_k8s: boolean;
    allow_docker: boolean; allow_team: boolean; is_active: boolean;
}
interface InvoiceEntry {
    id: string; plan_id: string; amount_cents: number; currency: string;
    status: string; paid_at: string; due_at: string; created_at: string;
}
interface Subscription {
    plan_name: string; plan_type: string; expires_at: string | null; stripe_subscription_id: string | null;
}

export default function Billing() {
    const toast = useToast();
    const [plans, setPlans] = useState<Plan[]>([]);
    const [invoices, setInvoices] = useState<InvoiceEntry[]>([]);
    const [subscription, setSubscription] = useState<Subscription | null>(null);
    const [loading, setLoading] = useState(true);
    const [checkoutLoading, setCheckoutLoading] = useState<string | null>(null);
    const [cancelLoading, setCancelLoading] = useState(false);
    const [showCancelConfirm, setShowCancelConfirm] = useState(false);

    useEffect(() => {
        Promise.all([
            billingService.listPlans(),
            billingService.listInvoices(),
            billingService.getSubscription().catch(() => null),
        ]).then(([p, i, s]) => {
            setPlans(Array.isArray(p) ? p : []);
            setInvoices(Array.isArray(i) ? i : []);
            setSubscription(s);
        }).catch(() => { setPlans([]); setInvoices([]); })
          .finally(() => setLoading(false));
    }, []);

    const handleSelectPlan = async (planId: string) => {
        setCheckoutLoading(planId);
        try {
            const { checkout_url } = await billingService.createCheckout(planId);
            window.location.href = checkout_url;
        } catch {
            toast.error('Failed to start checkout');
            setCheckoutLoading(null);
        }
    };

    const handleCancel = async () => {
        setCancelLoading(true);
        try {
            await billingService.cancelSubscription();
            toast.success('Subscription will cancel at period end');
            setShowCancelConfirm(false);
            setSubscription(s => s ? { ...s, stripe_subscription_id: null } : s);
        } catch { toast.error('Failed to cancel subscription'); }
        finally { setCancelLoading(false); }
    };

    if (loading) return (
        <div className="flex items-center justify-center py-20 animate-fade-in">
            <Loader className="w-6 h-6 text-nova-400 animate-spin" />
        </div>
    );

    const fmtDate = (s: string | null) => s ? new Date(s).toLocaleDateString() : '—';

    return (
        <div className="space-y-8 animate-fade-in">
            <div className="flex items-start justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <CreditCard className="w-7 h-7 text-nova-400" /> Billing & Plans
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Manage your subscription and view invoices</p>
                </div>
                {subscription && (
                    <div className="text-right">
                        <p className="text-sm font-medium text-white">{subscription.plan_name}</p>
                        <p className="text-xs text-surface-200/50 capitalize">{subscription.plan_type} plan
                            {subscription.expires_at && ` · renews ${fmtDate(subscription.expires_at)}`}
                        </p>
                        {subscription.stripe_subscription_id && (
                            <button onClick={() => setShowCancelConfirm(true)}
                                className="text-xs text-danger hover:text-danger/80 mt-1 transition-colors">
                                Cancel subscription
                            </button>
                        )}
                    </div>
                )}
            </div>

            {/* Plans */}
            {plans.length === 0 ? (
                <div className="glass-card rounded-2xl flex flex-col items-center justify-center py-16 text-surface-200/40">
                    <CreditCard className="w-10 h-10 mb-3 opacity-40" />
                    <p className="text-sm">No billing plans configured</p>
                    <p className="text-xs mt-1">Contact your administrator to set up billing</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {plans.map((plan, idx) => {
                        const isFeatured = plan.plan_type === 'enterprise';
                        const isCurrent = subscription?.plan_name === plan.name;
                        const isFree = plan.price_cents === 0;
                        return (
                            <div key={plan.id} className={`glass-card rounded-2xl p-6 relative overflow-hidden transition-all hover:border-white/10 ${isFeatured ? 'border-nova-500/30 ring-1 ring-nova-500/20' : ''}`}>
                                {isFeatured && (
                                    <div className="absolute top-0 right-0 px-3 py-1 rounded-bl-xl bg-gradient-to-r from-nova-500 to-nova-600 text-xs font-bold text-white flex items-center gap-1">
                                        <Zap className="w-3 h-3" /> POPULAR
                                    </div>
                                )}
                                {isCurrent && (
                                    <div className="absolute top-0 left-0 px-3 py-1 rounded-br-xl bg-success/20 text-xs font-bold text-success flex items-center gap-1">
                                        <Check className="w-3 h-3" /> CURRENT
                                    </div>
                                )}
                                <div className="mb-4 mt-2">
                                    <h3 className="text-lg font-bold text-white flex items-center gap-2">
                                        {plan.plan_type === 'enterprise' && <Crown className="w-4 h-4 text-warning" />}
                                        {plan.name}
                                    </h3>
                                    <div className="mt-2">
                                        <span className="text-3xl font-bold text-white">
                                            {isFree ? 'Free' : `$${(plan.price_cents / 100).toFixed(0)}`}
                                        </span>
                                        {!isFree && <span className="text-surface-200/40 text-sm">/mo</span>}
                                    </div>
                                </div>
                                <ul className="space-y-2 mb-6 text-sm">
                                    <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />{plan.max_servers} Server{plan.max_servers !== 1 ? 's' : ''}</li>
                                    <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />{plan.max_domains} Domains</li>
                                    <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />{plan.max_databases} Databases</li>
                                    <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />{plan.max_email} Email Accounts</li>
                                    <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />{plan.disk_gb} GB Storage</li>
                                    {plan.allow_waf && <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />WAF & Firewall</li>}
                                    {plan.allow_team && <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />Team Management</li>}
                                    {plan.allow_k8s && <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />Kubernetes</li>}
                                    {plan.allow_docker && <li className="flex items-center gap-2 text-surface-200/70"><Check className="w-3.5 h-3.5 text-success flex-shrink-0" />Docker</li>}
                                </ul>
                                {isCurrent ? (
                                    <div className="w-full py-2.5 rounded-xl text-sm font-medium text-center bg-success/10 text-success border border-success/20">
                                        Current Plan
                                    </div>
                                ) : isFree ? (
                                    <div className="w-full py-2.5 rounded-xl text-sm font-medium text-center border border-white/10 text-surface-200/40 cursor-default">
                                        Free Forever
                                    </div>
                                ) : (
                                    <button
                                        onClick={() => handleSelectPlan(plan.id)}
                                        disabled={!!checkoutLoading}
                                        className={`w-full py-2.5 rounded-xl text-sm font-medium transition-all flex items-center justify-center gap-2 ${isFeatured
                                            ? 'bg-gradient-to-r from-nova-600 to-nova-700 text-white hover:shadow-lg hover:shadow-nova-500/25'
                                            : 'border border-white/10 text-surface-200/60 hover:bg-white/5'
                                        } disabled:opacity-50`}>
                                        {checkoutLoading === plan.id ? <Loader className="w-4 h-4 animate-spin" /> : <ExternalLink className="w-4 h-4" />}
                                        Upgrade to {plan.name}
                                    </button>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}

            {/* Invoice History */}
            <div className="glass-card rounded-2xl overflow-hidden">
                <div className="px-5 py-3 border-b border-white/5 flex items-center gap-2">
                    <Receipt className="w-4 h-4 text-surface-200/40" />
                    <h3 className="text-sm font-semibold text-white">Invoice History</h3>
                </div>
                {invoices.length === 0 ? (
                    <div className="flex items-center justify-center py-12 text-surface-200/40 text-sm">No invoices yet</div>
                ) : (
                    <table className="w-full">
                        <thead><tr className="border-b border-white/5">
                            <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Invoice</th>
                            <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Amount</th>
                            <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Status</th>
                            <th className="text-left py-2 px-5 text-xs font-semibold text-surface-200/50">Date</th>
                        </tr></thead>
                        <tbody>
                            {invoices.map(inv => (
                                <tr key={inv.id} className="border-b border-white/5 hover:bg-white/[0.02] transition-colors">
                                    <td className="py-2.5 px-5 text-sm text-white font-mono">{inv.id.substring(0, 8)}</td>
                                    <td className="py-2.5 px-5 text-sm text-white font-medium">${(inv.amount_cents / 100).toFixed(2)}</td>
                                    <td className="py-2.5 px-5">
                                        <span className={`text-xs px-2 py-0.5 rounded-full border ${inv.status === 'paid' ? 'bg-success/10 text-success border-success/20' : 'bg-warning/10 text-warning border-warning/20'}`}>{inv.status}</span>
                                    </td>
                                    <td className="py-2.5 px-5 text-sm text-surface-200/40">{fmtDate(inv.created_at)}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>

            {/* Cancel Confirm Modal */}
            {showCancelConfirm && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowCancelConfirm(false)}>
                    <div className="glass-card rounded-2xl p-6 w-full max-w-md mx-4 space-y-4" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between">
                            <h3 className="text-lg font-bold text-white">Cancel Subscription</h3>
                            <button onClick={() => setShowCancelConfirm(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        <p className="text-sm text-surface-200/60">Your subscription will remain active until the end of the current billing period, then revert to the Community plan.</p>
                        <div className="flex gap-3 pt-2">
                            <button onClick={() => setShowCancelConfirm(false)} className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">Keep Plan</button>
                            <button onClick={handleCancel} disabled={cancelLoading}
                                className="flex-1 py-2.5 rounded-xl bg-danger text-white text-sm font-medium hover:bg-danger/80 transition-colors flex items-center justify-center gap-2 disabled:opacity-50">
                                {cancelLoading ? <Loader className="w-4 h-4 animate-spin" /> : null} Confirm Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
