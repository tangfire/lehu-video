import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { FiActivity, FiAlertCircle, FiArrowRight, FiBell, FiCpu, FiRefreshCw, FiSearch, FiSend, FiShield, FiZap } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { excerpt } from './adminUtils';
import './Admin.css';

const taskTypes = [
    { key: 'daily_ops', title: '每日巡检', icon: <FiActivity />, hint: '社区、风险、e仔、RAG' },
    { key: 'rag_gap', title: 'RAG缺口', icon: <FiSearch />, hint: '错误回复、低置信度、评测集' },
    { key: 'moderation_advice', title: '治理建议', icon: <FiShield />, hint: '审核、举报、反馈优先级' },
];

const riskText = {
    low: '低风险',
    medium: '中风险',
    high: '高风险',
};

const feishuText = {
    pending: '待发送',
    sent: '已发送',
    failed: '发送失败',
    skipped: '未发送',
};

const alertStatusText = {
    pending: '待发送',
    processing: '发送中',
    sent: '已发送',
    skipped: '已跳过',
    failed: '失败',
};

const alertTypeText = {
    report_created: '举报',
    feedback_important: '反馈',
    audit_review_required: '待审',
    audit_high_risk: '高风险',
    ai_budget_warning: '预算',
};

const AdminCopilot = () => {
    const [runType, setRunType] = useState('daily_ops');
    const [question, setQuestion] = useState('');
    const [runs, setRuns] = useState([]);
    const [current, setCurrent] = useState(null);
    const [feishu, setFeishu] = useState(null);
    const [opsAlerts, setOpsAlerts] = useState(null);
    const [loading, setLoading] = useState(false);
    const [sending, setSending] = useState(false);
    const [error, setError] = useState('');

    const loadRuns = useCallback(async () => {
        try {
            const [data, alertData] = await Promise.all([
                campusAdminApi.listCopilotRuns({ page: 1, size: 10 }),
                campusAdminApi.getCopilotOpsAlertsSummary(),
            ]);
            setRuns(data.runs || []);
            setFeishu(data.feishu || null);
            setOpsAlerts(alertData.summary || null);
            if (!current && (data.runs || []).length) setCurrent(data.runs[0]);
        } catch (err) {
            setError(err.message || '获取 Agent 记录失败');
        }
    }, [current]);

    useEffect(() => {
        loadRuns();
    }, []); // eslint-disable-line react-hooks/exhaustive-deps

    const selected = useMemo(() => taskTypes.find((item) => item.key === runType) || taskTypes[0], [runType]);
    const result = current?.result || {};

    const runCopilot = async () => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.createCopilotRun({ run_type: runType, question });
            setCurrent(data.run || null);
            await loadRuns();
        } catch (err) {
            setError(err.message || '运行 Agent 失败');
        } finally {
            setLoading(false);
        }
    };

    const sendFeishu = async () => {
        if (!current || current.status !== 'done') return;
        setSending(true);
        setError('');
        try {
            const data = await campusAdminApi.sendCopilotRunFeishu(current.id, {});
            const next = data.run || current;
            setCurrent(next);
            setRuns((items) => items.map((item) => (item.id === next.id ? next : item)));
            await loadRuns();
        } catch (err) {
            setError(err.message || '发送飞书失败');
        } finally {
            setSending(false);
        }
    };

    const feishuReady = Boolean(feishu?.enabled && feishu?.webhook_configured);
    const sendDisabled = !current || current.status !== 'done' || sending || current.feishu_status === 'sent' || !feishuReady;
    const sendText = sending ? '发送中' : current?.feishu_status === 'sent' ? '已发送飞书' : !feishuReady ? '飞书未配置' : '发送到飞书';
    const alertCounts = opsAlerts || {};
    const hasAlertFailures = Number(alertCounts.failed_count || 0) > 0 || Boolean(alertCounts.last_error);

    return (
        <div className="admin-copilot-page">
            {error && <div className="admin-error">{error}</div>}
            <section className="admin-copilot-hero">
                <div>
                    <span className="admin-kicker">LANGGRAPH AGENT</span>
                    <h2>值班 Agent</h2>
                    <p>巡检、举报反馈提醒、审核人工闭环。</p>
                </div>
                <div className={`admin-copilot-risk risk-${current?.risk_level || 'low'}`}>
                    <FiCpu />
                    <span>{riskText[current?.risk_level] || '未运行'}</span>
                </div>
            </section>

            <section className="admin-copilot-statusbar">
                <span>飞书：{feishu?.enabled ? '已开启' : '未开启'}</span>
                <span>日报：{feishu?.daily_enabled ? `${feishu.daily_time || '09:30'} 自动发送` : '未开启'}</span>
                <span>高风险：{feishu?.high_risk_enabled ? '即时提醒' : '未开启'}</span>
                <span>举报：{feishu?.report_notify_enabled ? '即时提醒' : '未开启'}</span>
                <span>反馈：{feishu?.feedback_notify_types || '未配置'}</span>
                <span>审核：{feishu?.audit_callback_enabled ? `按钮确认 · ${feishu.audit_auto_pass_confidence || '0.85'}` : '回后台处理'}</span>
                <span>Webhook：{feishu?.webhook_configured ? '已配置' : '未配置'}</span>
            </section>

            <section className={`admin-panel admin-ops-alert-panel ${hasAlertFailures ? 'has-failures' : ''}`}>
                <div className="admin-panel-head">
                    <div>
                        <h2>飞书提醒队列</h2>
                        <p>{alertCounts.last_sent_at ? `最近发送 ${alertCounts.last_sent_at}` : '暂无发送记录'}</p>
                    </div>
                    <button className="admin-button" type="button" onClick={loadRuns}><FiRefreshCw />刷新</button>
                </div>
                <div className="admin-ops-alert-stats">
                    <AlertStat icon={<FiBell />} label="待发送" value={alertCounts.pending_count || 0} />
                    <AlertStat icon={<FiRefreshCw />} label="发送中" value={alertCounts.processing_count || 0} />
                    <AlertStat icon={<FiAlertCircle />} label="失败" value={alertCounts.failed_count || 0} danger={hasAlertFailures} />
                    <AlertStat icon={<FiSend />} label="今日已发" value={alertCounts.sent_today_count || 0} />
                </div>
                {hasAlertFailures && (
                    <div className="admin-ops-alert-error">
                        <FiAlertCircle />
                        <span>{alertCounts.last_failed_at ? `${alertCounts.last_failed_at} · ` : ''}{alertCounts.last_error || '有提醒发送失败'}</span>
                    </div>
                )}
                <div className="admin-ops-alert-list">
                    {(alertCounts.recent_alerts || []).map((item) => (
                        <article className={`admin-ops-alert-row ${item.status === 'failed' || item.feishu_status === 'failed' ? 'failed' : ''}`} key={item.id}>
                            <strong>{alertTypeText[item.alert_type] || item.alert_type} · {item.title || '值班提醒'}</strong>
                            <span>{excerpt(item.summary || item.feishu_error || '', 96)}</span>
                            <em>{alertStatusText[item.status] || item.status} · 重试 {item.retry_count || 0} · {item.created_at}</em>
                        </article>
                    ))}
                    {!(alertCounts.recent_alerts || []).length && <div className="admin-empty compact">暂无提醒记录</div>}
                </div>
            </section>

            <section className="admin-panel">
                <div className="admin-copilot-task-grid">
                    {taskTypes.map((item) => (
                        <button className={`admin-copilot-task ${runType === item.key ? 'active' : ''}`} key={item.key} type="button" onClick={() => setRunType(item.key)}>
                            {item.icon}
                            <strong>{item.title}</strong>
                            <span>{item.hint}</span>
                        </button>
                    ))}
                </div>
                <div className="admin-copilot-runbar">
                    <input className="admin-input" value={question} onChange={(e) => setQuestion(e.target.value)} placeholder={`补充关注点：例如 帮我看一下${selected.title}有没有异常`} />
                    <button className="admin-button primary" type="button" onClick={runCopilot} disabled={loading}>
                        {loading ? <FiRefreshCw className="spin" /> : <FiZap />}
                        运行 Agent
                    </button>
                </div>
            </section>

            <div className="admin-copilot-grid">
                <section className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <h2>分析结果</h2>
                            <p>{current ? `${current.run_type} · ${current.updated_at}` : '暂无运行记录'}</p>
                        </div>
                    </div>
                    {!current && <div className="admin-empty compact">选择任务后运行 Agent</div>}
                    {current && (
                        <div className="admin-copilot-result">
                            <div className={`admin-copilot-summary risk-${current.risk_level || 'low'}`}>
                                <strong>{result.summary || current.summary || '暂无摘要'}</strong>
                                <span>{riskText[result.risk_level || current.risk_level] || '低风险'}</span>
                            </div>
                            <div className="admin-copilot-meta">
                                <span>{current.source === 'scheduled' ? '自动日报' : '手动运行'}</span>
                                <span className={`admin-copilot-feishu ${current.feishu_status || 'pending'}`}>
                                    飞书：{feishuText[current.feishu_status] || current.feishu_status || '待发送'}
                                </span>
                                {current.feishu_sent_at && <span>{current.feishu_sent_at}</span>}
                            </div>
                            <CopilotList title="关键发现" items={result.findings} kind="finding" />
                            <CopilotList title="建议动作" items={result.recommendations} kind="recommendation" />
                            <div className="admin-copilot-actions">
                                <button className="admin-button primary" type="button" onClick={sendFeishu} disabled={sendDisabled}>
                                    {sending ? <FiRefreshCw className="spin" /> : <FiSend />}
                                    {sendText}
                                </button>
                                {(result.next_actions || []).map((item) => (
                                    <Link className="admin-button" to={item.path || '/admin'} key={`${item.label}-${item.path}`}>
                                        {item.label}
                                        <FiArrowRight />
                                    </Link>
                                ))}
                            </div>
                            <CopilotList title="证据来源" items={result.evidence} kind="evidence" />
                        </div>
                    )}
                </section>

                <aside className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <h2>运行记录</h2>
                            <p>最近 10 次。</p>
                        </div>
                        <button className="admin-button" type="button" onClick={loadRuns}><FiRefreshCw />刷新</button>
                    </div>
                    <div className="admin-copilot-runs">
                        {!runs.length && <div className="admin-empty compact">暂无记录</div>}
                        {runs.map((item) => (
                            <button className={`admin-copilot-run ${current?.id === item.id ? 'active' : ''}`} key={item.id} type="button" onClick={() => setCurrent(item)}>
                                <strong>{excerpt(item.summary || item.run_type, 42)}</strong>
                                <span>{riskText[item.risk_level] || item.risk_level} · {feishuText[item.feishu_status] || item.feishu_status || '待发送'} · {item.created_at}</span>
                            </button>
                        ))}
                    </div>
                    <div className="admin-copilot-trace">
                        <strong>工具调用</strong>
                        {(current?.tool_trace || []).map((item) => (
                            <div key={item.tool}>
                                <span>{item.tool}</span>
                                <em>{item.ok ? 'ok' : 'failed'} · {item.duration_ms || 0}ms</em>
                            </div>
                        ))}
                        {!(current?.tool_trace || []).length && <p>暂无 trace</p>}
                    </div>
                </aside>
            </div>
        </div>
    );
};

const AlertStat = ({ icon, label, value, danger }) => (
    <div className={`admin-ops-alert-stat ${danger ? 'danger' : ''}`}>
        {icon}
        <span>{label}</span>
        <strong>{value}</strong>
    </div>
);

const CopilotList = ({ title, items = [], kind }) => (
    <div className={`admin-copilot-list ${kind}`}>
        <h3>{title}</h3>
        {!items.length && <p>暂无</p>}
        {items.map((item, index) => (
            <article key={`${title}-${index}`}>
                <strong>{item.title || item.source || item.label || '未命名'}</strong>
                <span>{item.detail || item.priority || item.severity || item.link || ''}</span>
            </article>
        ))}
    </div>
);

export default AdminCopilot;
