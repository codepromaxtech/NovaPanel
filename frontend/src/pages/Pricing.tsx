import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Zap, Check, X, ArrowRight, Loader, Shield, Server, Globe, Activity, ChevronDown, ChevronUp } from 'lucide-react';
import axios from 'axios';

const LICENSE_SERVER = 'https://license.codepromax.com.de';

// ─── Plan definitions (mirrors the license server) ───────────────────────────

const plans = [
    {
        id: 'community',
        name: 'Community',
        badge: null,
        description: 'Perfect for personal projects and learning',
        monthly: 0,
        yearly: 0,
        color: 'border-surface-700/40',
        accent: 'text-surface-200/60',
        buttonClass: 'bg-surface-700 hover:bg-surface-600 text-white',
        highlights: [
            '1 Server',
            '3 Domains',
            '2 Databases',
            '10 Email Accounts',
            'SSH Terminal',
            'File Manager',
            'Basic Monitoring',
            'Community Support',
        ],
        locked: [
            'WAF & Firewall',
            'Cloudflare Tunnels',
            'Docker & Kubernetes',
            'Wildcard SSL',
            'Team Management',
            'API Keys',
            'FTP/SFTP',
            'Multi-Server Deploy',
        ],
    },
    {
        id: 'professional',
        name: 'Professional',
        badge: null,
        description: 'For growing teams and small agencies',
        monthly: 19,
        yearly: 190,
        color: 'border-nova-500/40',
        accent: 'text-nova-400',
        buttonClass: 'bg-nova-600 hover:bg-nova-500 text-white shadow-lg shadow-nova-500/20',
        highlights: [
            '10 Servers',
            '30 Domains',
            '20 Databases',
            '200 Email Accounts',
            'WAF & Firewall',
            'Cloudflare Tunnels',
            'Docker Support',
            'API Keys & Webhooks',
            'FTP/SFTP Accounts',
            'Email Support',
        ],
        locked: [
            'Kubernetes (k3s)',
            'Wildcard SSL',
            'Team Management',
            'Reseller System',
            'Multi-Server Deploy',
        ],
    },
    {
        id: 'enterprise',
        name: 'Enterprise',
        badge: 'Most Popular',
        description: 'Full power for professional infrastructure',
        monthly: 49,
        yearly: 490,
        color: 'border-purple-500/50',
        accent: 'text-purple-400',
        buttonClass: 'bg-gradient-to-r from-purple-600 to-nova-600 hover:from-purple-500 hover:to-nova-500 text-white shadow-lg shadow-purple-500/20',
        highlights: [
            'Unlimited Servers & Domains',
            'Unlimited Databases',
            'Full WAF + Firewall',
            'Cloudflare Tunnels',
            'Docker & Kubernetes',
            'Wildcard SSL (DNS challenge)',
            'Team Management + RBAC',
            'API Keys & Webhooks',
            'FTP/SFTP Accounts',
            'Multi-Server Deploy',
            'Priority Email Support',
        ],
        locked: [
            'Reseller Client Management',
        ],
    },
    {
        id: 'reseller',
        name: 'Reseller',
        badge: 'White-label',
        description: 'Run a hosting business on your brand',
        monthly: 99,
        yearly: 990,
        color: 'border-yellow-500/40',
        accent: 'text-yellow-400',
        buttonClass: 'bg-gradient-to-r from-yellow-600 to-orange-600 hover:from-yellow-500 hover:to-orange-500 text-white shadow-lg shadow-yellow-500/20',
        highlights: [
            'Everything in Enterprise',
            'Reseller Client Accounts',
            'Per-client Resource Quotas',
            'Bulk Domain Provisioning',
            'White-label Ready',
            'Dedicated Support Channel',
        ],
        locked: [],
    },
];

const features = [
    { label: 'Servers',            community: '1',   professional: '10',      enterprise: 'Unlimited', reseller: 'Unlimited' },
    { label: 'Domains',            community: '3',   professional: '30',      enterprise: 'Unlimited', reseller: 'Unlimited' },
    { label: 'Databases',          community: '2',   professional: '20',      enterprise: 'Unlimited', reseller: 'Unlimited' },
    { label: 'Email Accounts',     community: '10',  professional: '200',     enterprise: 'Unlimited', reseller: 'Unlimited' },
    { label: 'SSH Terminal',       community: true,  professional: true,      enterprise: true,        reseller: true },
    { label: 'File Manager',       community: true,  professional: true,      enterprise: true,        reseller: true },
    { label: 'WAF & Firewall',     community: false, professional: true,      enterprise: true,        reseller: true },
    { label: 'Docker',             community: false, professional: true,      enterprise: true,        reseller: true },
    { label: 'Kubernetes',         community: false, professional: false,     enterprise: true,        reseller: true },
    { label: 'Cloudflare Tunnels', community: false, professional: true,      enterprise: true,        reseller: true },
    { label: 'Wildcard SSL',       community: false, professional: false,     enterprise: true,        reseller: true },
    { label: 'Team Management',    community: false, professional: false,     enterprise: true,        reseller: true },
    { label: 'API Keys',           community: false, professional: true,      enterprise: true,        reseller: true },
    { label: 'FTP/SFTP',           community: false, professional: true,      enterprise: true,        reseller: true },
    { label: 'Multi-Server Deploy',community: false, professional: false,     enterprise: true,        reseller: true },
    { label: 'Reseller System',    community: false, professional: false,     enterprise: false,       reseller: true },
    { label: 'Support',            community: 'Community', professional: 'Email', enterprise: 'Priority', reseller: 'Dedicated' },
];

// ─── Checkout modal ───────────────────────────────────────────────────────────

function CheckoutModal({ plan, cycle, onClose }: { plan: typeof plans[0]; cycle: 'monthly' | 'yearly'; onClose: () => void }) {
    const [email, setEmail] = useState('');
    const [paymentMethod, setPaymentMethod] = useState<'sslcommerz' | 'bkash'>('sslcommerz');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const price = cycle === 'yearly' ? plan.yearly : plan.monthly;

    const handlePurchase = async () => {
        if (!email.match(/^[^\s@]+@[^\s@]+\.[^\s@]+$/)) { setError('Enter a valid email address'); return; }
        setLoading(true); setError('');
        try {
            const res = await axios.post(`${LICENSE_SERVER}/index.php?route=purchase&action=init`, {
                email, plan: plan.id, cycle, payment_method: paymentMethod,
            });
            const data = res.data;
            if (data.success && data.checkout_url) {
                window.location.href = data.checkout_url;
            } else {
                setError('Payment gateway error. Please try again.');
            }
        } catch (e: any) {
            setError(e?.response?.data?.error || 'Failed to initiate payment. Please try again.');
        } finally { setLoading(false); }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={onClose}>
            <div className="glass-card rounded-2xl p-8 w-full max-w-md mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                <div className="flex items-center justify-between mb-6">
                    <div>
                        <h2 className="text-xl font-bold text-white">Get {plan.name}</h2>
                        <p className="text-sm text-surface-200/50 mt-0.5">
                            ${price}/{cycle === 'yearly' ? 'year' : 'month'}
                            {cycle === 'yearly' && <span className="ml-2 text-green-400 text-xs">2 months free</span>}
                        </p>
                    </div>
                    <button onClick={onClose} className="text-surface-200/40 hover:text-white transition-colors">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Your Email</label>
                        <input type="email" value={email} onChange={e => setEmail(e.target.value)}
                            className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                            placeholder="you@company.com" />
                        <p className="text-xs text-surface-200/40 mt-1">Your license key will be sent here after payment.</p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-surface-200 mb-1.5">Payment Method</label>
                        <div className="grid grid-cols-2 gap-3">
                            {[
                                { id: 'sslcommerz', label: '💳 Card / Bank', sub: 'Visa, Mastercard, Mobile Banking' },
                                { id: 'bkash',      label: '📱 bKash',       sub: 'Mobile wallet (Bangladesh)' },
                            ].map(m => (
                                <button key={m.id} type="button"
                                    onClick={() => setPaymentMethod(m.id as any)}
                                    className={`p-3 rounded-xl border text-left transition-all ${paymentMethod === m.id ? 'border-nova-500/50 bg-nova-500/10' : 'border-surface-700/40 hover:bg-white/5'}`}>
                                    <p className="text-sm font-medium text-white">{m.label}</p>
                                    <p className="text-[10px] text-surface-200/40 mt-0.5">{m.sub}</p>
                                </button>
                            ))}
                        </div>
                    </div>

                    {error && (
                        <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/30 text-red-300 text-sm">{error}</div>
                    )}

                    <div className="p-3 rounded-xl bg-surface-800/50 border border-surface-700/30 text-xs text-surface-200/50 space-y-1">
                        <p>• After payment, your license key is emailed to you instantly.</p>
                        <p>• Paste the key in <strong className="text-surface-200/80">Settings → License → Activate</strong>.</p>
                        <p>• Subscriptions renew automatically. Cancel any time.</p>
                    </div>

                    <button onClick={handlePurchase} disabled={loading || !email}
                        className={`w-full py-3 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all disabled:opacity-50 ${plan.buttonClass}`}>
                        {loading ? <Loader className="w-4 h-4 animate-spin" /> : <ArrowRight className="w-4 h-4" />}
                        {loading ? 'Redirecting to payment...' : `Pay $${price} → Get License Key`}
                    </button>
                </div>
            </div>
        </div>
    );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function Pricing() {
    const navigate = useNavigate();
    const [cycle, setCycle] = useState<'monthly' | 'yearly'>('monthly');
    const [checkoutPlan, setCheckoutPlan] = useState<typeof plans[0] | null>(null);
    const [showTable, setShowTable] = useState(false);

    const Cell = ({ val }: { val: string | boolean }) => {
        if (val === true)  return <Check className="w-4 h-4 text-green-400 mx-auto" />;
        if (val === false) return <X className="w-4 h-4 text-surface-200/20 mx-auto" />;
        return <span className="text-sm text-surface-200/70">{val as string}</span>;
    };

    return (
        <div className="min-h-screen bg-surface-950 text-white font-sans">
            {/* Nav */}
            <nav className="border-b border-white/5 px-6 py-4 flex items-center justify-between max-w-7xl mx-auto">
                <button onClick={() => navigate('/login')} className="flex items-center gap-3">
                    <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-nova-500 to-nova-700 flex items-center justify-center">
                        <Zap className="w-5 h-5 text-white" />
                    </div>
                    <span className="text-lg font-bold">Nova<span className="text-nova-400">Panel</span></span>
                </button>
                <button onClick={() => navigate('/login')}
                    className="flex items-center gap-2 px-4 py-2 rounded-xl border border-white/10 text-sm text-surface-200/60 hover:text-white hover:border-white/20 transition-all">
                    Sign in <ArrowRight className="w-3.5 h-3.5" />
                </button>
            </nav>

            {/* Hero */}
            <div className="text-center py-20 px-4 max-w-4xl mx-auto">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-nova-500/30 bg-nova-500/10 mb-6">
                    <span className="w-1.5 h-1.5 rounded-full bg-nova-400 animate-pulse" />
                    <span className="text-xs font-semibold text-nova-300 uppercase tracking-wider">Simple, transparent pricing</span>
                </div>
                <h1 className="text-5xl font-extrabold mb-5 tracking-tight">
                    The right plan for<br />
                    <span className="bg-gradient-to-r from-nova-400 to-purple-400 bg-clip-text text-transparent">your infrastructure</span>
                </h1>
                <p className="text-xl text-surface-200/60 max-w-2xl mx-auto mb-10">
                    Start free, scale when you're ready. Every paid plan includes a 15-day free trial — no credit card required to start.
                </p>

                {/* Billing cycle toggle */}
                <div className="inline-flex items-center gap-1 p-1 rounded-xl bg-surface-800/60 border border-white/10">
                    {(['monthly', 'yearly'] as const).map(c => (
                        <button key={c} onClick={() => setCycle(c)}
                            className={`px-5 py-2 rounded-lg text-sm font-medium transition-all ${cycle === c ? 'bg-nova-600 text-white shadow-lg' : 'text-surface-200/50 hover:text-white'}`}>
                            {c === 'monthly' ? 'Monthly' : 'Yearly'}
                            {c === 'yearly' && <span className="ml-2 text-[10px] font-bold text-green-400 bg-green-400/10 px-1.5 py-0.5 rounded-full">-17%</span>}
                        </button>
                    ))}
                </div>
            </div>

            {/* Plan cards */}
            <div className="max-w-7xl mx-auto px-4 pb-20">
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-5">
                    {plans.map(plan => {
                        const price  = cycle === 'yearly' ? plan.yearly : plan.monthly;
                        const isFree = plan.id === 'community';
                        return (
                            <div key={plan.id}
                                className={`relative flex flex-col rounded-2xl border bg-surface-900/60 backdrop-blur p-6 ${plan.color} ${plan.badge ? 'ring-1 ring-purple-500/30' : ''}`}>
                                {plan.badge && (
                                    <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider bg-gradient-to-r from-purple-600 to-nova-600 text-white shadow-lg">
                                        {plan.badge}
                                    </div>
                                )}

                                <div className="mb-5">
                                    <h2 className={`text-lg font-bold mb-0.5 ${plan.accent}`}>{plan.name}</h2>
                                    <p className="text-xs text-surface-200/40">{plan.description}</p>
                                </div>

                                <div className="mb-6">
                                    {isFree ? (
                                        <div>
                                            <span className="text-4xl font-extrabold text-white">Free</span>
                                            <p className="text-xs text-surface-200/40 mt-1">Forever</p>
                                        </div>
                                    ) : (
                                        <div>
                                            <div className="flex items-end gap-1">
                                                <span className="text-4xl font-extrabold text-white">${price}</span>
                                                <span className="text-surface-200/40 text-sm mb-1">/{cycle === 'yearly' ? 'yr' : 'mo'}</span>
                                            </div>
                                            {cycle === 'yearly' && (
                                                <p className="text-xs text-green-400 mt-1">${(plan.monthly * 12 - plan.yearly).toFixed(0)} saved vs monthly</p>
                                            )}
                                        </div>
                                    )}
                                </div>

                                <button
                                    onClick={() => isFree ? navigate('/login') : setCheckoutPlan(plan)}
                                    className={`w-full py-2.5 rounded-xl text-sm font-semibold flex items-center justify-center gap-2 transition-all mb-6 ${plan.buttonClass}`}>
                                    {isFree ? 'Get Started Free' : `Get ${plan.name}`}
                                    <ArrowRight className="w-3.5 h-3.5" />
                                </button>

                                <div className="space-y-2.5 flex-1">
                                    {plan.highlights.map(h => (
                                        <div key={h} className="flex items-start gap-2">
                                            <Check className="w-3.5 h-3.5 text-green-400 flex-shrink-0 mt-0.5" />
                                            <span className="text-xs text-surface-200/70">{h}</span>
                                        </div>
                                    ))}
                                    {plan.locked.map(l => (
                                        <div key={l} className="flex items-start gap-2 opacity-40">
                                            <X className="w-3.5 h-3.5 text-surface-200/50 flex-shrink-0 mt-0.5" />
                                            <span className="text-xs text-surface-200/40 line-through">{l}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        );
                    })}
                </div>

                {/* Trial notice */}
                <div className="mt-10 text-center">
                    <div className="inline-flex items-center gap-3 px-6 py-3 rounded-2xl bg-surface-800/50 border border-white/5 text-sm text-surface-200/60">
                        <Shield className="w-4 h-4 text-nova-400" />
                        All new installs get a <strong className="text-white">15-day Enterprise trial</strong> automatically — no card needed.
                    </div>
                </div>

                {/* Feature comparison table */}
                <div className="mt-16">
                    <button onClick={() => setShowTable(t => !t)}
                        className="flex items-center gap-2 mx-auto text-sm text-surface-200/50 hover:text-white transition-colors mb-6">
                        {showTable ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                        {showTable ? 'Hide' : 'Show'} full feature comparison
                    </button>

                    {showTable && (
                        <div className="overflow-x-auto rounded-2xl border border-white/5">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="border-b border-white/5 bg-surface-900/80">
                                        <th className="text-left px-5 py-4 text-surface-200/40 font-medium w-48">Feature</th>
                                        {plans.map(p => (
                                            <th key={p.id} className={`px-4 py-4 font-semibold text-center ${p.accent}`}>{p.name}</th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    {features.map((row, i) => (
                                        <tr key={row.label} className={`border-b border-white/5 ${i % 2 === 0 ? 'bg-surface-900/20' : ''}`}>
                                            <td className="px-5 py-3 text-surface-200/60">{row.label}</td>
                                            <td className="px-4 py-3 text-center"><Cell val={row.community} /></td>
                                            <td className="px-4 py-3 text-center"><Cell val={row.professional} /></td>
                                            <td className="px-4 py-3 text-center"><Cell val={row.enterprise} /></td>
                                            <td className="px-4 py-3 text-center"><Cell val={row.reseller} /></td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>

                {/* FAQ / trust signals */}
                <div className="mt-20 grid grid-cols-1 md:grid-cols-3 gap-6 text-center">
                    {[
                        { icon: Shield, title: '30-day refund', body: 'Not happy? Get a full refund within 30 days, no questions asked.' },
                        { icon: Server, title: 'Self-hosted', body: 'Your data stays on your server. No SaaS lock-in, ever.' },
                        { icon: Activity, title: 'Instant upgrade', body: 'Paste your license key in Settings and features unlock immediately.' },
                    ].map(({ icon: Icon, title, body }) => (
                        <div key={title} className="p-6 rounded-2xl border border-white/5 bg-surface-900/40">
                            <Icon className="w-6 h-6 text-nova-400 mx-auto mb-3" />
                            <h3 className="text-sm font-semibold text-white mb-2">{title}</h3>
                            <p className="text-xs text-surface-200/50">{body}</p>
                        </div>
                    ))}
                </div>

                {/* Footer */}
                <div className="mt-12 text-center border-t border-white/5 pt-8">
                    <p className="text-xs text-surface-200/30">
                        © {new Date().getFullYear()} NovaPanel · Powered by CodeProMax ·{' '}
                        <button onClick={() => navigate('/login')} className="text-nova-400 hover:text-nova-300 transition-colors">Sign In</button>
                    </p>
                </div>
            </div>

            {checkoutPlan && (
                <CheckoutModal plan={checkoutPlan} cycle={cycle} onClose={() => setCheckoutPlan(null)} />
            )}
        </div>
    );
}
