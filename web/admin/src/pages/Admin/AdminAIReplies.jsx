import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { FiAlertCircle, FiCheckCircle, FiCpu, FiExternalLink, FiRefreshCw, FiShield, FiTrash2 } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { compactNumber, excerpt } from './adminUtils';
import './Admin.css';

const pageSize = 20;

const statusText = (status) => {
    const map = {
        pending: '待处理',
        processing: '处理中',
        done: '已回复',
        failed: '失败',
    };
    return map[status] || '全部';
};

const qualityOptions = [
    ['good', '好'],
    ['needs_fix', '待优化'],
    ['wrong', '错误'],
    ['unsafe', '风险'],
];

const qualityText = (label) => {
    const item = qualityOptions.find(([value]) => value === label);
    return item ? item[1] : '未标注';
};

const AdminAIReplies = ({ mode = 'full', initialStatus = 'failed' }) => {
    const [summary, setSummary] = useState(null);
    const [tasks, setTasks] = useState([]);
    const [status, setStatus] = useState(initialStatus);
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [retrying, setRetrying] = useState('');
    const [reviewing, setReviewing] = useState('');
    const [withdrawing, setWithdrawing] = useState('');
    const [savingSettings, setSavingSettings] = useState(false);
    const [error, setError] = useState('');
    const [toast, setToast] = useState('');

    const load = useCallback(async (nextPage = page, nextStatus = status) => {
        setLoading(true);
        setError('');
        try {
            const [summaryData, taskData] = await Promise.all([
                campusAdminApi.aiReplySummary(),
                campusAdminApi.listAiReplyTasks({ status: nextStatus, page: nextPage, size: pageSize }),
            ]);
            setSummary(summaryData.summary || {});
            setTasks(taskData.tasks || []);
            setTotal(taskData.page_stats?.total || 0);
            setPage(nextPage);
        } catch (err) {
            setError(err.message || '获取 e仔回复任务失败');
        } finally {
            setLoading(false);
        }
    }, [page, status]);

    useEffect(() => {
        load(1, status);
    }, [status]); // eslint-disable-line react-hooks/exhaustive-deps

    const stats = useMemo(() => {
        if (!summary) return [];
        return [
            { label: '今日回复', value: summary.today_used || 0, hint: `上限 ${compactNumber(summary.daily_limit || 0)}` },
            { label: '待处理', value: summary.pending || 0, hint: '等待后台任务领取' },
            { label: '处理中', value: summary.processing || 0, hint: '超时会自动重试' },
            { label: '失败', value: summary.failed || 0, hint: '可手动重新加入队列' },
        ];
    }, [summary]);

    const handleStatus = (nextStatus) => {
        setStatus(nextStatus);
        setPage(1);
    };

    const handleRetry = async (id) => {
        if (retrying) return;
        setRetrying(id);
        setToast('');
        setError('');
        try {
            await campusAdminApi.retryAiReplyTask(id);
            setToast('已重新加入回复队列');
            await load(page, status);
        } catch (err) {
            setError(err.message || '重试失败');
        } finally {
            setRetrying('');
        }
    };

    const handleWithdraw = async (task) => {
        if (withdrawing || !task?.answer_comment_id || task.answer_comment_id === '0') return;
        if (!window.confirm('确定撤回这条 e仔回复吗？撤回后评论区将不可见。')) return;
        setWithdrawing(task.id);
        setToast('');
        setError('');
        try {
            await campusAdminApi.moderateAiReplyTask(task.id, { action: 'withdraw' });
            setToast('已撤回 e仔回复');
            await load(page, status);
        } catch (err) {
            setError(err.message || '撤回失败');
        } finally {
            setWithdrawing('');
        }
    };

    const handleQuality = async (task, label) => {
        const logID = task?.rag_log?.id;
        if (!logID || reviewing) return;
        setReviewing(logID);
        setToast('');
        setError('');
        try {
            await campusAdminApi.reviewKnowledgeQueryLog(logID, { label, note: task.rag_log?.quality_note || '' });
            setToast(`已标注为：${qualityText(label)}`);
            await load(page, status);
        } catch (err) {
            setError(err.message || '标注失败');
        } finally {
            setReviewing('');
        }
    };

    const handleAutoReplyToggle = async () => {
        if (savingSettings || !summary) return;
        const nextEnabled = !summary.auto_reply_enabled;
        setSavingSettings(true);
        setToast('');
        setError('');
        try {
            const data = await campusAdminApi.updateAiReplySettings({ auto_reply_enabled: nextEnabled });
            setSummary(data.summary || {});
            setToast(nextEnabled ? 'e仔自动回复已开启' : 'e仔自动回复已关闭，@e仔 将通知官方账号');
        } catch (err) {
            setError(err.message || '保存 e仔设置失败');
        } finally {
            setSavingSettings(false);
        }
    };

    return (
        <div className="admin-ai-page">
            {error && <div className="admin-error">{error}</div>}
            {toast && <div className="admin-toast success">{toast}</div>}

            {mode === 'full' && (
                <section className="admin-ops-toolbar">
                    <div>
                        <span className="admin-kicker">@深汕e仔</span>
                        <strong>回复状态 · 失败任务 · 手动重试</strong>
                    </div>
                    <button className="admin-button" type="button" onClick={() => load(page, status)} disabled={loading}>
                        <FiRefreshCw className={loading ? 'spin' : ''} />
                        刷新
                    </button>
                </section>
            )}

            <section className="admin-ai-health">
                <div className={`admin-ai-status ${summary?.bot_ready ? 'ok' : 'off'}`}>
                    {summary?.bot_ready ? <FiCheckCircle /> : <FiAlertCircle />}
                    <div className="admin-ai-avatar">
                        {summary?.bot_avatar
                            ? <img src={summary.bot_avatar} alt="" />
                            : <span>{(summary?.bot_name || 'e').slice(0, 1).toUpperCase()}</span>}
                    </div>
                    <div>
                        <strong>{summary?.bot_ready ? '官方账号已配置' : '官方账号待确认'}</strong>
                        <span>
                            {summary?.bot_user_id
                                ? `${summary?.bot_name || '未找到用户昵称'} · 用户 ${summary?.bot_user_id}`
                                : '需要配置 CAMPUS_EZAI_BOT_USER_ID'}
                        </span>
                    </div>
                </div>
                <div className="admin-ai-note">
                    <FiCpu />
                    <span>{summary?.auto_reply_enabled ? '@e仔 评论触发，必要时先查知识库。' : '自动回复已关，@e仔 将通知官方账号人工处理。'}</span>
                </div>
                <div className={`admin-ai-status ${summary?.effective_enabled ? 'ok' : 'off'}`}>
                    {summary?.effective_enabled ? <FiCheckCircle /> : <FiAlertCircle />}
                    <div>
                        <strong>{summary?.auto_reply_enabled ? (summary?.effective_enabled ? '自动回复已开启' : '自动回复待配置') : '自动回复已关闭'}</strong>
                        <span>{summary?.model_configured ? `${summary?.model || '-'} · 今日 ${compactNumber(summary?.today_used || 0)}/${compactNumber(summary?.daily_limit || 0)}` : '需要配置模型 key 和官方账号'}</span>
                    </div>
                    <button className="admin-button subtle" type="button" disabled={savingSettings || loading} onClick={handleAutoReplyToggle}>
                        {savingSettings ? '保存中...' : (summary?.auto_reply_enabled ? '关闭' : '开启')}
                    </button>
                </div>
                <div className={`admin-ai-status ${summary?.rag_health?.status === 'ok' ? 'ok' : 'off'}`}>
                    {summary?.rag_health?.status === 'ok' ? <FiCheckCircle /> : <FiAlertCircle />}
                    <div>
                        <strong>RAG：{summary?.rag_health?.status || '未知'}</strong>
                        <span>
                            Qdrant {summary?.rag_health?.qdrant || '-'} · 知识片段 {compactNumber(summary?.rag_health?.chunk_count || 0)}
                            {summary?.rag_health?.last_error ? ` · ${summary.rag_health.last_error}` : ''}
                        </span>
                    </div>
                </div>
            </section>

            <section className="admin-key-grid ai">
                {stats.map((item) => (
                    <div className="admin-key-stat" key={item.label}>
                        <span>{item.label}</span>
                        <strong>{compactNumber(item.value)}</strong>
                        <em>{item.hint}</em>
                    </div>
                ))}
            </section>

            {mode !== 'summary' && <section className="admin-panel">
                <div className="admin-panel-head">
                    <div>
                        <h2>回复任务</h2>
                        <p>失败优先。</p>
                    </div>
                    <div className="admin-segment">
                        {['failed', 'pending', 'processing', 'done', ''].map((item) => (
                            <button
                                key={item || 'all'}
                                className={status === item ? 'active' : ''}
                                type="button"
                                onClick={() => handleStatus(item)}
                            >
                                {statusText(item)}
                            </button>
                        ))}
                    </div>
                </div>

                {loading && !tasks.length ? <div className="admin-loading">任务加载中...</div> : (
                    <div className="admin-ai-task-list">
                        {!tasks.length && <div className="admin-empty compact">暂无{statusText(status)}任务</div>}
                        {tasks.map((task) => (
                            <article className="admin-ai-task" key={task.id}>
                                <div className="admin-ai-task-main">
                                    <div className="admin-ai-task-head">
                                        <span className={`admin-status ai-status-${task.status}`}>{statusText(task.status)}</span>
                                        <span>任务 {task.id}</span>
                                        <span>重试 {task.retry_count || 0}</span>
                                        <span>{task.updated_at || task.created_at}</span>
                                    </div>
                                    <p>{excerpt(task.prompt, 110) || '无提问内容'}</p>
                                    <div className="admin-ai-quality-grid">
                                        <div>
                                            <span>原评论</span>
                                            <p>{excerpt(task.trigger_comment?.content, 180) || '未找到触发评论'}</p>
                                        </div>
                                        <div>
                                            <span>e仔回复</span>
                                            <p>{excerpt(task.answer_comment?.content || task.rag_log?.answer, 220) || (task.status === 'done' ? '未找到回复内容' : '尚未生成')}</p>
                                        </div>
                                    </div>
                                    {task.rag_log && (
                                        <div className="admin-ai-rag-card">
                                            <div className="admin-ai-rag-head">
                                                <span>RAG {task.rag_log.need_knowledge ? '需要知识库' : '无需知识库'} · 置信度 {Number(task.rag_log.confidence || 0).toFixed(2)} · {task.rag_log.duration_ms || 0}ms</span>
                                                <strong>{qualityText(task.rag_log.quality_label)}</strong>
                                            </div>
                                            {!!task.rag_log.hit_chunks?.length && (
                                                <div className="admin-ai-rag-chunks">
                                                    {task.rag_log.hit_chunks.slice(0, 3).map((chunk) => (
                                                        <span key={`${task.id}-${chunk.chunk_id || chunk.document_id}-${chunk.score}`}>
                                                            {chunk.title || chunk.source || '知识片段'} {Number(chunk.score || 0).toFixed(2)}
                                                        </span>
                                                    ))}
                                                </div>
                                            )}
                                            {task.rag_log.error_message && <div className="admin-ai-error">{task.rag_log.error_message}</div>}
                                            <div className="admin-ai-quality-actions">
                                                {qualityOptions.map(([value, label]) => (
                                                    <button
                                                        key={value}
                                                        className={`admin-button subtle ${task.rag_log.quality_label === value ? 'active' : ''}`}
                                                        type="button"
                                                        onClick={() => handleQuality(task, value)}
                                                        disabled={reviewing === task.rag_log.id}
                                                    >
                                                        <FiShield />
                                                        {label}
                                                    </button>
                                                ))}
                                            </div>
                                        </div>
                                    )}
                                    {task.last_error && <div className="admin-ai-error">{task.last_error}</div>}
                                    <div className="admin-ai-task-meta">
                                        <Link to={`/admin/posts?keyword=${task.post_id}`}>
                                            帖子 {task.post_id} <FiExternalLink />
                                        </Link>
                                        <span>触发评论 {task.trigger_comment_id}</span>
                                        {task.answer_comment_id && task.answer_comment_id !== '0' && <span>回复评论 {task.answer_comment_id}</span>}
                                        {task.next_retry_at && <span>下次重试 {task.next_retry_at}</span>}
                                    </div>
                                </div>
                                <div className="admin-ai-task-actions">
                                    {task.answer_comment_id && task.answer_comment_id !== '0' && (
                                        <button className="admin-button danger" type="button" onClick={() => handleWithdraw(task)} disabled={withdrawing === task.id}>
                                            <FiTrash2 />
                                            撤回
                                        </button>
                                    )}
                                    {(task.status === 'failed' || task.status === 'processing') && (
                                        <button className="admin-button" type="button" onClick={() => handleRetry(task.id)} disabled={retrying === task.id}>
                                            <FiRefreshCw className={retrying === task.id ? 'spin' : ''} />
                                            重试
                                        </button>
                                    )}
                                </div>
                            </article>
                        ))}
                    </div>
                )}

                <div className="admin-pagination">
                    <span>共 {compactNumber(total)} 条</span>
                    <button className="admin-button" disabled={loading || page <= 1} onClick={() => load(page - 1, status)}>上一页</button>
                    <button className="admin-button" disabled={loading || page * pageSize >= total} onClick={() => load(page + 1, status)}>下一页</button>
                </div>
            </section>}
        </div>
    );
};

export default AdminAIReplies;
