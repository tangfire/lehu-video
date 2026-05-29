import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FiCheckCircle, FiExternalLink, FiMessageCircle, FiTrash2 } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { statusText } from './adminUtils';
import './Admin.css';

const AdminComments = () => {
    const [searchParams] = useSearchParams();
    const statusParam = searchParams.get('status') || '-1';
    const postIDParam = searchParams.get('post_id') || '';
    const [comments, setComments] = useState([]);
    const [status, setStatus] = useState(statusParam);
    const [postID, setPostID] = useState(postIDParam);
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [error, setError] = useState('');
    const [message, setMessage] = useState('');
    const [loading, setLoading] = useState(false);
    const [actionLoading, setActionLoading] = useState(false);
    const [pendingDelete, setPendingDelete] = useState(null);

    const load = async (nextPage = page, nextStatus = status, nextPostID = postID) => {
        setLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.listComments({ page: nextPage, size: 20, status: nextStatus, post_id: nextPostID });
            setComments(data.comments || []);
            setTotal(data.page_stats?.total || 0);
            setPage(nextPage);
        } catch (err) {
            setError(err.message || '获取评论失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        setStatus(statusParam);
        setPostID(postIDParam);
        load(1, statusParam, postIDParam);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [statusParam, postIDParam]);

    const showMessage = (text) => {
        setMessage(text);
        window.setTimeout(() => setMessage(''), 2400);
    };

    const remove = async () => {
        if (!pendingDelete) return;
        setActionLoading(true);
        setError('');
        try {
            await campusAdminApi.deleteComment(pendingDelete.id);
            setPendingDelete(null);
            showMessage('评论已下架');
            load(page);
        } catch (err) {
            setError(err.message || '下架评论失败');
        } finally {
            setActionLoading(false);
        }
    };

    const reviewComment = async (comment, action) => {
        setActionLoading(true);
        setError('');
        try {
            await campusAdminApi.reviewComment(comment.id, {
                action,
                reason: action === 'visible' ? '后台恢复可见' : '后台下架评论',
            });
            showMessage(action === 'visible' ? '评论已恢复可见' : '评论已下架');
            load(page);
        } catch (err) {
            setError(err.message || '评论状态更新失败');
        } finally {
            setActionLoading(false);
        }
    };

    return (
        <>
            {message && <div className="admin-toast success">{message}</div>}
            <div className="admin-toolbar">
                <select className="admin-select" value={status} onChange={(e) => setStatus(e.target.value)}>
                    <option value="-1">全部状态</option>
                    <option value="1">正常展示</option>
                    <option value="0">待审核</option>
                    <option value="2">已拒绝</option>
                    <option value="3">已下架</option>
                </select>
                <input className="admin-input" value={postID} onChange={(e) => setPostID(e.target.value)} placeholder="按帖子 ID 筛选，可留空" />
                <button className="admin-button primary" onClick={() => load(1)}>查询</button>
            </div>
            {error && <div className="admin-error">{error}</div>}
            {loading && <div className="admin-loading">评论加载中...</div>}
            {!loading && comments.length === 0 && (
                <div className="admin-empty">
                    <FiMessageCircle />
                    <span>暂无评论数据</span>
                </div>
            )}
            {!loading && comments.length > 0 && <div className="admin-table-wrap">
                <table className="admin-table">
                    <thead>
                        <tr>
                            <th>评论</th>
                            <th>关联帖子</th>
                            <th>用户</th>
                            <th>状态</th>
                            <th>时间</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        {comments.map((comment) => (
                            <tr key={comment.id}>
                                <td className="admin-title-cell">{comment.content}</td>
                                <td>{comment.post?.title || comment.post_id}</td>
                                <td>{comment.author?.name || comment.author?.nickname || '同学'}</td>
                                <td>{statusText(comment.status)}</td>
                                <td>{comment.created_at}</td>
                                <td>
                                    <div className="admin-actions">
                                        {comment.post_id && (
                                            <button className="admin-button" type="button" onClick={() => window.open(`/admin/posts?keyword=${encodeURIComponent(String(comment.post_id))}`, '_blank', 'noopener,noreferrer')}>
                                                <FiExternalLink />
                                                原帖
                                            </button>
                                        )}
                                        {Number(comment.status) === 1 ? (
                                            <button className="admin-button danger" onClick={() => setPendingDelete(comment)} disabled={actionLoading}>
                                                <FiTrash2 />
                                                下架
                                            </button>
                                        ) : (
                                            <button className="admin-button" disabled={actionLoading} onClick={() => reviewComment(comment, 'visible')}>
                                                <FiCheckCircle />
                                                恢复可见
                                            </button>
                                        )}
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>}
            <div className="admin-pagination">
                <span className="admin-muted">共 {total} 条</span>
                <button className="admin-button" disabled={page <= 1} onClick={() => load(page - 1)}>上一页</button>
                <button className="admin-button" disabled={page * 20 >= total} onClick={() => load(page + 1)}>下一页</button>
            </div>
            {pendingDelete && (
                <div className="admin-modal-backdrop" role="presentation">
                    <div className="admin-confirm-modal">
                        <div className="admin-modal-icon"><FiTrash2 /></div>
                        <h3>下架评论</h3>
                        <p>确认下架这条评论吗？下架后前台将不再展示。</p>
                        <div className="admin-modal-actions">
                            <button className="admin-button" onClick={() => setPendingDelete(null)}>取消</button>
                            <button className="admin-button danger" disabled={actionLoading} onClick={remove}>确认下架</button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};

export default AdminComments;
