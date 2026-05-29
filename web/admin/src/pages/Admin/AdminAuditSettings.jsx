import { useEffect, useMemo, useState } from 'react';
import { FiAlertTriangle, FiCheckCircle, FiClock, FiRefreshCw, FiShield, FiZap } from 'react-icons/fi';
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
        title: 'AI 初审',
        badge: '提效',
        summary: 'AI 先判，运营复核',
        detail: '适合发帖量变大后。',
    },
];

const modeTitle = (mode) => auditModes.find((item) => item.value === mode)?.title || '不审核';

const AdminAuditSettings = () => {
    const [settings, setSettings] = useState(null);
    const [selectedMode, setSelectedMode] = useState('off');
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');

    const load = async () => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.getAuditSettings();
            const next = data.settings || {};
            setSettings(next);
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
    const aiReady = Boolean(settings?.ai_enabled);

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
                        {aiReady ? 'AI 可用' : 'AI 未配置'}
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
                        <span>AI 未配置时，新帖会保留在待审核队列。</span>
                    </div>
                )}
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

export default AdminAuditSettings;
