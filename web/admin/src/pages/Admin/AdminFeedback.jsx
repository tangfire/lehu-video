import { useEffect, useState } from 'react';
import { FiSend } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import './Admin.css';

const statusText = (status) => {
    const map = { 0: '待处理', 1: '处理中', 2: '已处理' };
    return map[Number(status)] || '未知';
};

const typeText = (type) => {
    const map = {
        bug: '问题反馈',
        suggestion: '功能建议',
        content: '内容纠错',
        cooperation: '合作投稿',
        contact: '联系我们',
    };
    return map[type] || '功能建议';
};

const AdminFeedback = () => {
    const [items, setItems] = useState([]);
    const [status, setStatus] = useState('-1');
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');
    const [loading, setLoading] = useState(false);

    const load = async (nextPage = page, nextStatus = status) => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listFeedback({ page: nextPage, size: 20, status: nextStatus });
            setItems(data.feedbacks || []);
            setTotal(data.page_stats?.total || 0);
            setPage(nextPage);
        } catch (err) {
            setError(err.message || '获取反馈失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load(1, status);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const review = async (item, nextStatus) => {
        const note = nextStatus === 2 ? '已处理' : '正在跟进';
        try {
            await campusAdminApi.reviewFeedback(item.id, { status: nextStatus, operator_note: note });
            setMessage(nextStatus === 2 ? '反馈已设为已处理' : '反馈已设为处理中');
            window.setTimeout(() => setMessage(''), 2400);
            load(page);
        } catch (err) {
            setError(err.message || '处理反馈失败');
        }
    };

    return (
        <>
            {message && <div className="admin-toast success">{message}</div>}
            <div className="admin-toolbar">
                <select className="admin-select" value={status} onChange={(e) => setStatus(e.target.value)}>
                    <option value="-1">全部反馈</option>
                    <option value="0">待处理</option>
                    <option value="1">处理中</option>
                    <option value="2">已处理</option>
                </select>
                <button className="admin-button primary" onClick={() => load(1)}>查询</button>
            </div>
            {error && <div className="admin-error">{error}</div>}
            {loading && <div className="admin-loading">反馈加载中...</div>}
            {!loading && items.length === 0 && (
                <div className="admin-empty">
                    <FiSend />
                    <span>暂无用户反馈</span>
                </div>
            )}
            {!loading && items.length > 0 && (
                <div className="admin-feedback-list">
                    {items.map((item) => (
                        <article className="admin-feedback-card" key={item.id}>
                            <div className="admin-feedback-main">
                                <div className="admin-feedback-head">
                                    <span className="admin-chip">{typeText(item.feedback_type)}</span>
                                    <span className={`admin-status status-${item.status}`}>{statusText(item.status)}</span>
                                    <span className="admin-muted">{item.created_at}</span>
                                </div>
                                <p>{item.content}</p>
                                <div className="admin-muted">
                                    {item.author?.name || item.author?.nickname || '同学'} · {item.contact || '未留联系方式'}
                                </div>
                                {item.operator_note && <div className="admin-feedback-note">备注：{item.operator_note}</div>}
                                {!!item.images?.length && (
                                    <div className="admin-feedback-images">
                                        {item.images.map((image) => (
                                            <a key={image} href={image} target="_blank" rel="noreferrer">
                                                <img src={image} alt="反馈截图" />
                                            </a>
                                        ))}
                                    </div>
                                )}
                            </div>
                            <div className="admin-feedback-actions">
                                <button className="admin-button" disabled={Number(item.status) === 1} onClick={() => review(item, 1)}>设为处理中</button>
                                <button className="admin-button primary" disabled={Number(item.status) === 2} onClick={() => review(item, 2)}>设为已处理</button>
                            </div>
                        </article>
                    ))}
                </div>
            )}
            <div className="admin-pagination">
                <span className="admin-muted">共 {total} 条</span>
                <button className="admin-button" disabled={page <= 1} onClick={() => load(page - 1)}>上一页</button>
                <button className="admin-button" disabled={page * 20 >= total} onClick={() => load(page + 1)}>下一页</button>
            </div>
        </>
    );
};

export default AdminFeedback;
