import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { FiActivity, FiArrowRight, FiCpu, FiRefreshCw, FiSearch, FiShield, FiZap } from 'react-icons/fi';
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

const AdminCopilot = () => {
    const [runType, setRunType] = useState('daily_ops');
    const [question, setQuestion] = useState('');
    const [runs, setRuns] = useState([]);
    const [current, setCurrent] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const loadRuns = useCallback(async () => {
        try {
            const data = await campusAdminApi.listCopilotRuns({ page: 1, size: 10 });
            setRuns(data.runs || []);
            if (!current && (data.runs || []).length) setCurrent(data.runs[0]);
        } catch (err) {
            setError(err.message || '获取 Copilot 记录失败');
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
            setError(err.message || '运行 Copilot 失败');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="admin-copilot-page">
            {error && <div className="admin-error">{error}</div>}
            <section className="admin-copilot-hero">
                <div>
                    <span className="admin-kicker">LANGGRAPH AGENT</span>
                    <h2>运营 Copilot</h2>
                    <p>只读分析，工具调用，运营确认后再处理。</p>
                </div>
                <div className={`admin-copilot-risk risk-${current?.risk_level || 'low'}`}>
                    <FiCpu />
                    <span>{riskText[current?.risk_level] || '未运行'}</span>
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
                        运行 Copilot
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
                    {!current && <div className="admin-empty compact">选择任务后运行 Copilot</div>}
                    {current && (
                        <div className="admin-copilot-result">
                            <div className={`admin-copilot-summary risk-${current.risk_level || 'low'}`}>
                                <strong>{result.summary || current.summary || '暂无摘要'}</strong>
                                <span>{riskText[result.risk_level || current.risk_level] || '低风险'}</span>
                            </div>
                            <CopilotList title="关键发现" items={result.findings} kind="finding" />
                            <CopilotList title="建议动作" items={result.recommendations} kind="recommendation" />
                            <div className="admin-copilot-actions">
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
                                <span>{riskText[item.risk_level] || item.risk_level} · {item.created_at}</span>
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
