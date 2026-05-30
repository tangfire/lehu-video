import { useEffect, useMemo, useState } from 'react';
import { FiAlertTriangle, FiBell, FiCheckCircle, FiClock, FiCpu, FiRefreshCw, FiShield, FiZap } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import './Admin.css';

const auditModes = [
    {
        value: 'off',
        title: '不审核',
        badge: '开放',
        summary: '新帖直接展示',
        detail: '适合内测冷启动。',
    },
    {
        value: 'manual',
        title: '人工审核',
        badge: '稳妥',
        summary: '新帖进入待审核',
        detail: '适合正式推广期。',
    },
    {
        value: 'ai',
        title: 'AI/Agent 初审',
        badge: '提效',
        summary: '低风险自动过',
        detail: '不确定进飞书/后台。',
    },
];

const modeTitle = (mode) => auditModes.find((item) => item.value === mode)?.title || '不审核';

const agentSwitches = [
    { key: 'agent_enabled', title: 'Agent 模型能力', detail: '控制 Copilot、日报、AI 初审等模型调用' },
    { key: 'agent_audit_enabled', title: 'AI/Agent 初审', detail: '关闭后 AI 模式会退化为人工待审' },
    { key: 'feishu_ops_enabled', title: '飞书运营通知', detail: '总开关，控制值班提醒是否发群' },
    { key: 'daily_report_enabled', title: '每日报告', detail: '按配置时间生成运营日报' },
    { key: 'high_risk_notify_enabled', title: '高风险提醒', detail: 'Agent 判断高风险时即时提醒' },
    { key: 'report_notify_enabled', title: '举报提醒', detail: '用户举报后推送飞书' },
    { key: 'feedback_notify_enabled', title: '重要反馈提醒', detail: '联系我们、Bug、内容问题即时提醒' },
];

const normalizeAgentSettings = (settings = {}) => ({
    agent_enabled: Boolean(settings.agent_enabled),
    agent_audit_enabled: Boolean(settings.agent_audit_enabled),
    feishu_ops_enabled: Boolean(settings.feishu_ops_enabled),
    daily_report_enabled: Boolean(settings.daily_report_enabled),
    high_risk_notify_enabled: Boolean(settings.high_risk_notify_enabled),
    report_notify_enabled: Boolean(settings.report_notify_enabled),
    feedback_notify_enabled: Boolean(settings.feedback_notify_enabled),
    webhook_configured: Boolean(settings.webhook_configured),
    public_api_base_url_configured: Boolean(settings.public_api_base_url_configured),
    agent_service_configured: Boolean(settings.agent_service_configured),
    agent_model_configured: Boolean(settings.agent_model_configured),
    updated_at: settings.updated_at || '',
});

const AdminAuditSettings = () => {
    const [settings, setSettings] = useState(null);
    const [agentSettings, setAgentSettings] = useState(null);
    const [agentDraft, setAgentDraft] = useState(normalizeAgentSettings());
    const [selectedMode, setSelectedMode] = useState('off');
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [agentSaving, setAgentSaving] = useState(false);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');

    const load = async () => {
        setLoading(true);
        setError('');
        try {
            const [auditData, agentData] = await Promise.all([
                campusAdminApi.getAuditSettings(),
                campusAdminApi.getAgentSettings(),
            ]);
            const next = auditData.settings || {};
            const nextAgent = normalizeAgentSettings(agentData.settings || {});
            setSettings(next);
            setAgentSettings(nextAgent);
            setAgentDraft(nextAgent);
            setSelectedMode(next.post_audit_mode || 'off');
        } catch (err) {
            setError(err.message || '获取审核设置失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load();
    }, []);

    const currentMode = settings?.post_audit_mode || 'off';
    const changed = selectedMode !== currentMode;
    const selected = useMemo(() => auditModes.find((item) => item.value === selectedMode) || auditModes[0], [selectedMode]);
    const aiReady = Boolean(settings?.ai_enabled && agentDraft.agent_enabled && agentDraft.agent_audit_enabled);
    const agentChanged = useMemo(() => Boolean(agentSettings) && JSON.stringify(agentDraft) !== JSON.stringify(agentSettings), [agentDraft, agentSettings]);

    const requestSave = () => {
        if (!changed || saving) return;
        setConfirmOpen(true);
    };

    const save = async () => {
        setSaving(true);
        setError('');
        setMessage('');
        try {
            const data = await campusAdminApi.updateAuditSettings({ post_audit_mode: selectedMode });
            const next = data.settings || {};
            setSettings(next);
            setSelectedMode(next.post_audit_mode || selectedMode);
            setConfirmOpen(false);
            setMessage('审核设置已保存');
            window.setTimeout(() => setMessage(''), 2400);
        } catch (err) {
            setError(err.message || '保存审核设置失败');
        } finally {
            setSaving(false);
        }
    };

    const saveAgent = async () => {
        setAgentSaving(true);
        setError('');
        setMessage('');
        try {
            const data = await campusAdminApi.updateAgentSettings({
                agent_enabled: agentDraft.agent_enabled,
                agent_audit_enabled: agentDraft.agent_audit_enabled,
                feishu_ops_enabled: agentDraft.feishu_ops_enabled,
                daily_report_enabled: agentDraft.daily_report_enabled,
                high_risk_notify_enabled: agentDraft.high_risk_notify_enabled,
                report_notify_enabled: agentDraft.report_notify_enabled,
                feedback_notify_enabled: agentDraft.feedback_notify_enabled,
            });
            const next = normalizeAgentSettings(data.settings || {});
            setAgentSettings(next);
            setAgentDraft(next);
            setMessage('Agent 设置已保存');
            window.setTimeout(() => setMessage(''), 2400);
        } catch (err) {
            setError(err.message || '保存 Agent 设置失败');
        } finally {
            setAgentSaving(false);
        }
    };

    const toggleAgentDraft = (key) => {
        setAgentDraft((prev) => ({ ...prev, [key]: !prev[key] }));
    };

    return (
        <div className="admin-audit-settings-page">
            {message && <div className="admin-toast success">{message}</div>}
            {error && <div className="admin-error">{error}</div>}

            <section className="admin-audit-ops-panel">
                <div className="admin-audit-header">
                    <div>
                        <span className="admin-kicker">OPS SETTINGS</span>
                        <h2>发帖审核</h2>
                    </div>
                    <button className="admin-icon-button" type="button" disabled={loading} onClick={load} title="刷新">
                        <FiRefreshCw className={loading ? 'spin' : ''} />
                    </button>
                </div>

                <div className={`admin-audit-status-strip mode-${currentMode}`}>
                    <FiShield />
                    <div>
                        <strong>当前模式：{modeTitle(currentMode)}</strong>
                        <span>{currentMode === 'off' ? '新帖直接展示' : '新帖先进入审核队列'}</span>
                    </div>
                    <span className={`admin-ai-pill ${aiReady ? 'ready' : 'missing'}`}>
                        {aiReady ? <FiZap /> : <FiAlertTriangle />}
                        {aiReady ? 'Agent 可用' : 'Agent 未配置'}
                    </span>
                </div>

                <div className="admin-audit-segmented" role="radiogroup" aria-label="发帖审核模式">
                    {auditModes.map((mode) => (
                        <button
                            className={`admin-audit-segment ${selectedMode === mode.value ? 'active' : ''}`}
                            type="button"
                            key={mode.value}
                            role="radio"
                            aria-checked={selectedMode === mode.value}
                            onClick={() => setSelectedMode(mode.value)}
                        >
                            <span className="admin-audit-segment-top">
                                <strong>{mode.title}</strong>
                                <em>{mode.badge}</em>
                            </span>
                            <span>{mode.summary}</span>
                            <small>{mode.detail}</small>
                        </button>
                    ))}
                </div>

                <div className="admin-audit-footer">
                    <div className="admin-audit-meta">
                        <FiClock />
                        <span>{settings?.updated_at ? `上次更新 ${settings.updated_at}` : '暂无更新时间'}</span>
                    </div>
                    <div className="admin-audit-actions">
                        <span>{changed ? `待保存：${selected.title}` : '已是最新设置'}</span>
                        <button className="admin-button primary" type="button" disabled={!changed || saving} onClick={requestSave}>
                            {saving ? '保存中...' : '保存'}
                        </button>
                    </div>
                </div>

                {selectedMode === 'ai' && !aiReady && (
                    <div className="admin-audit-warning">
                        <FiAlertTriangle />
                        <span>Agent 或 AI 初审关闭时，新帖会保留在待审核队列。</span>
                    </div>
                )}
            </section>

            <section className="admin-audit-ops-panel admin-agent-settings-panel">
                <div className="admin-audit-header">
                    <div>
                        <span className="admin-kicker">DUTY AGENT</span>
                        <h2>值班 Agent 开关</h2>
                    </div>
                    <button className="admin-button primary" type="button" disabled={!agentChanged || agentSaving} onClick={saveAgent}>
                        {agentSaving ? '保存中...' : '保存开关'}
                    </button>
                </div>

                <div className="admin-agent-status-grid">
                    <StatusChip icon={<FiBell />} ok={agentDraft.feishu_ops_enabled && agentDraft.webhook_configured} label="飞书 Webhook" />
                    <StatusChip icon={<FiShield />} ok={agentDraft.public_api_base_url_configured} label="回调 URL" />
                    <StatusChip icon={<FiCpu />} ok={agentDraft.agent_service_configured} label="Agent 服务" />
                    <StatusChip icon={<FiZap />} ok={agentDraft.agent_model_configured} label="模型 Key" />
                </div>

                <div className="admin-agent-switch-grid">
                    {agentSwitches.map((item) => (
                        <label className={`admin-agent-switch ${agentDraft[item.key] ? 'on' : ''}`} key={item.key}>
                            <input type="checkbox" checked={agentDraft[item.key]} onChange={() => toggleAgentDraft(item.key)} />
                            <span>
                                <strong>{item.title}</strong>
                                <em>{item.detail}</em>
                            </span>
                        </label>
                    ))}
                </div>

                <div className="admin-audit-footer">
                    <div className="admin-audit-meta">
                        <FiClock />
                        <span>{agentSettings?.updated_at ? `上次更新 ${agentSettings.updated_at}` : '暂无更新时间'}</span>
                    </div>
                    <div className="admin-audit-actions">
                        <span>{agentChanged ? '有未保存开关' : '已是最新设置'}</span>
                    </div>
                </div>
            </section>

            {confirmOpen && (
                <div className="admin-modal-backdrop" role="presentation">
                    <div className="admin-confirm-modal compact">
                        <div className="admin-modal-icon danger"><FiAlertTriangle /></div>
                        <h3>保存审核模式</h3>
                        <p>确认将发帖审核切换为「{selected.title}」？</p>
                        <div className="admin-modal-actions">
                            <button className="admin-button" disabled={saving} onClick={() => setConfirmOpen(false)}>取消</button>
                            <button className="admin-button danger" disabled={saving} onClick={save}>
                                <FiCheckCircle />
                                确认保存
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

const StatusChip = ({ icon, ok, label }) => (
    <span className={`admin-agent-status-chip ${ok ? 'ok' : 'missing'}`}>
        {icon}
        {label}：{ok ? '正常' : '未配置'}
    </span>
);

export default AdminAuditSettings;
