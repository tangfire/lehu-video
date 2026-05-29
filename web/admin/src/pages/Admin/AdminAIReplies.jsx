import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { FiAlertCircle, FiCheckCircle, FiCpu, FiExternalLink, FiRefreshCw } from 'react-icons/fi';
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

const AdminAIReplies = ({ mode = 'full', initialStatus = 'failed' }) => {
    const [summary, setSummary] = useState(null);
    const [tasks, setTasks] = useState([]);
    const [status, setStatus] = useState(initialStatus);
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const [retrying, setRetrying] = useState('');
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

    return (
        <div className="admin-ai-page">
            {error && <div className="admin-error">{error}</div>}
            {toast && <div className="admin-toast success">{toast}</div>}

            {mode === 'full' && (
                <section className="admin-simple-head ai">
                    <div>
                        <span className="admin-kicker">评论区小彩蛋</span>
                        <h2>@深汕e仔 自动回复</h2>
                        <p>这里不配置模型密钥，只看链路状态、失败原因，并把失败任务重新加入队列。</p>
                    </div>
                    <button className="admin-button" type="button" onClick={() => load(page, status)} disabled={loading}>
                        <FiRefreshCw className={loading ? 'spin' : ''} />
                        刷新
                    </button>
                </section>
            )}

            <section className="admin-ai-health">
                <div className={`admin-ai-status ${summary?.enabled && summary?.bot_ready ? 'ok' : 'off'}`}>
                    {summary?.enabled && summary?.bot_ready ? <FiCheckCircle /> : <FiAlertCircle />}
                    <div>
                        <strong>{summary?.enabled ? (summary?.bot_ready ? '已启用' : '已启用，Bot 账号待确认') : '未启用'}</strong>
                        <span>
                            {summary?.enabled
                                ? `${summary?.bot_name || '未找到用户昵称'} · Bot 用户 ${summary?.bot_user_id || '-'} · ${summary?.model || '-'}`
                                : '需要配置 CAMPUS_AI_API_KEY / DEEPSEEK_API_KEY 和 CAMPUS_EZAI_BOT_USER_ID'}
                        </span>
                    </div>
                </div>
                <div className="admin-ai-note">
                    <FiCpu />
                    <span>用户评论里 @深汕e仔 后，后端会落任务表；需要校园事实时会先查 e仔知识库，再生成官方账号回复。</span>
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
                        <p>默认看失败任务，排查最省时间。</p>
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
