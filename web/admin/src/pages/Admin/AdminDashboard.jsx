import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { FiArrowRight, FiChevronDown, FiCpu, FiEdit3, FiFileText, FiFlag, FiMessageCircle, FiRefreshCw, FiSend, FiStar } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import { compactNumber, excerpt, postCover, postTypeText, ratioText } from './adminUtils';
import './Admin.css';

const AdminDashboard = () => {
    const [summary, setSummary] = useState(null);
    const [hotPosts, setHotPosts] = useState([]);
    const [aiSummary, setAiSummary] = useState(null);
    const [error, setError] = useState('');
    const [reconcileLoading, setReconcileLoading] = useState(false);
    const [reconcileResult, setReconcileResult] = useState(null);
    const [showAdvanced, setShowAdvanced] = useState(false);

    useEffect(() => {
        let mounted = true;
        Promise.all([
            campusAdminApi.summary(),
            campusAdminApi.listPosts({ sort: 'hot', status: 1, size: 5 }),
            campusAdminApi.aiReplySummary(),
        ])
            .then(([summaryData, postsData, aiData]) => {
                if (!mounted) return;
                setSummary(summaryData.summary || {});
                setHotPosts(postsData.posts || []);
                setAiSummary(aiData.summary || {});
            })
            .catch((err) => {
                if (mounted) setError(err.message || '获取数据失败');
            });
        return () => {
            mounted = false;
        };
    }, []);

    const keyStats = useMemo(() => {
        if (!summary) return [];
        const todayInteraction = Number(summary.today_comments || 0) + Number(summary.today_likes || 0) + Number(summary.today_collections || 0);
        const pending = Number(summary.pending_reports || 0) + Number(summary.pending_posts || 0) + Number(summary.pending_comments || 0) + Number(summary.pending_feedback || 0);
        return [
            { label: '今日访问', value: summary.today_visits || 0, hint: `累计 ${compactNumber(summary.total_visits)}` },
            { label: '今日登录', value: summary.today_logins || 0, hint: `新增用户 ${compactNumber(summary.today_users)}` },
            { label: '今日互动', value: todayInteraction, hint: '评论 / 点赞 / 收藏' },
            { label: '发布转化', value: ratioText(summary.today_publish_done, summary.today_publish_open), hint: `${compactNumber(summary.today_publish_open)} 次打开发布` },
            { label: '详情访问', value: summary.today_detail_views || 0, hint: '今日帖子详情' },
            { label: '待处理', value: pending, hint: '反馈 / 举报 / 审核' },
        ];
    }, [summary]);

    const todos = useMemo(() => {
        if (!summary) return [];
        const items = [];
        if (Number(summary.pending_reports || 0) > 0) {
            items.push({ icon: <FiFlag />, title: '处理举报', detail: `${summary.pending_reports} 条举报等待处理`, to: '/admin/moderation?tab=reports&status=0' });
        }
        if (Number(summary.pending_feedback || 0) > 0) {
            items.push({ icon: <FiSend />, title: '回复用户反馈', detail: `${summary.pending_feedback} 条反馈等待跟进`, to: '/admin/moderation?tab=feedback&status=0' });
        }
        if (Number(summary.pending_posts || 0) > 0) {
            items.push({ icon: <FiFileText />, title: '审核帖子', detail: `${summary.pending_posts} 篇帖子待审核`, to: '/admin/posts?status=0' });
        }
        if (Number(summary.pending_comments || 0) > 0) {
            items.push({ icon: <FiMessageCircle />, title: '审核评论', detail: `${summary.pending_comments} 条评论待处理`, to: '/admin/moderation?tab=comments&status=0' });
        }
        if (Number(summary.today_posts || 0) < 3) {
            items.push({ icon: <FiEdit3 />, title: '补 3 条官方内容', detail: '报到攻略、宿舍 FAQ、问答引导最适合今天发', to: '/admin/compose' });
        }
        if (Number(aiSummary?.failed || 0) > 0) {
            items.push({ icon: <FiCpu />, title: '处理 e仔回复失败', detail: `${aiSummary.failed} 条自动回复任务失败`, to: '/admin/assistant?tab=failed' });
        }
        return items.length > 0 ? items : [{ icon: <FiStar />, title: '今天状态不错', detail: '没有紧急待办，可以继续准备种子内容', to: '/admin/compose' }];
    }, [summary, aiSummary]);

    const handleReconcileStats = async () => {
        if (reconcileLoading) return;
        setReconcileLoading(true);
        setError('');
        try {
            const data = await campusAdminApi.reconcileStats();
            setReconcileResult(data.result || {});
            const summaryData = await campusAdminApi.summary();
            setSummary(summaryData.summary || {});
        } catch (err) {
            setError(err.message || '计数对账失败');
        } finally {
            setReconcileLoading(false);
        }
    };

    if (error) return <div className="admin-error">{error}</div>;
    if (!summary) return <div className="admin-loading">数据加载中...</div>;

    return (
        <div className="admin-dashboard simple">
            <section className="admin-simple-head">
                <div>
                    <span className="admin-kicker">今日运营</span>
                    <h2>先看待办，再发内容</h2>
                    <p>开学前后台只需要盯住内容供给、互动和审核风险。</p>
                </div>
                <div className="admin-head-actions">
                    <Link className="admin-button primary" to="/admin/compose">
                        <FiEdit3 />
                        发官方帖
                    </Link>
                </div>
            </section>

            <section className="admin-key-grid daily">
                {keyStats.map((item) => (
                    <div className="admin-key-stat" key={item.label}>
                        <span>{item.label}</span>
                        <strong>{compactNumber(item.value)}</strong>
                        <em>{item.hint}</em>
                    </div>
                ))}
            </section>

            <section className="admin-two-column simple">
                <div className="admin-panel">
                    <div className="admin-panel-head">
                        <h2>今天先做这些</h2>
                    </div>
                    <div className="admin-todo-list simple">
                        {todos.map((item) => (
                            <Link className="admin-todo-item" to={item.to} key={item.title}>
                                {item.icon}
                                <div>
                                    <strong>{item.title}</strong>
                                    <span>{item.detail}</span>
                                </div>
                                <FiArrowRight />
                            </Link>
                        ))}
                    </div>
                </div>

                <div className="admin-panel">
                    <div className="admin-panel-head">
                        <h2>风险提醒</h2>
                    </div>
                    <div className="admin-risk-list">
                        <Link className="admin-risk-row" to="/admin/moderation?tab=reports&status=0">
                            <strong>{compactNumber(summary.pending_reports || 0)}</strong>
                            <span>待处理举报</span>
                        </Link>
                        <Link className="admin-risk-row" to="/admin/moderation?tab=feedback&status=0">
                            <strong>{compactNumber(summary.pending_feedback || 0)}</strong>
                            <span>待跟进反馈</span>
                        </Link>
                        <Link className="admin-risk-row" to="/admin/posts?status=0">
                            <strong>{compactNumber(summary.pending_posts || 0)}</strong>
                            <span>待审核帖子</span>
                        </Link>
                        <Link className="admin-risk-row" to="/admin/assistant?tab=failed">
                            <strong>{compactNumber((aiSummary?.failed || 0) + (summary.pending_ai_audits || 0))}</strong>
                            <span>AI / e仔任务</span>
                        </Link>
                    </div>
                </div>
            </section>

            <section className="admin-dashboard-advanced-toggle">
                <button className="admin-button subtle" type="button" onClick={() => setShowAdvanced((value) => !value)}>
                    {showAdvanced ? '收起高级信息' : '展开高级信息'}
                    <FiChevronDown className={showAdvanced ? 'rotate' : ''} />
                </button>
            </section>

            {showAdvanced && <>
                <section className="admin-funnel-strip">
                    <div>
                        <span>发布成功</span>
                        <strong>{compactNumber(summary.today_publish_done)}</strong>
                    </div>
                    <div>
                        <span>今日分享</span>
                        <strong>{compactNumber(summary.today_shares)}</strong>
                    </div>
                    <div>
                        <span>用户反馈</span>
                        <strong>{compactNumber(summary.today_feedback)}</strong>
                    </div>
                    <div>
                        <span>新增举报</span>
                        <strong>{compactNumber(summary.today_reports)}</strong>
                    </div>
                </section>

                <section className="admin-ops-note">
                    <div>
                        <strong>互动计数对账</strong>
                        <span>从真实点赞、收藏、评论、回复关系表重新聚合，修正列表冗余计数。</span>
                    </div>
                    {reconcileResult ? (
                        <em>
                            已修正 {compactNumber(reconcileResult.updated_posts || 0)} 篇帖子、
                            {compactNumber(reconcileResult.updated_comments || 0)} 条评论
                        </em>
                    ) : (
                        <em>建议发布高峰后手动跑一次。</em>
                    )}
                    <button className="admin-button" type="button" onClick={handleReconcileStats} disabled={reconcileLoading}>
                        <FiRefreshCw className={reconcileLoading ? 'spin' : ''} />
                        计数对账
                    </button>
                </section>

                {aiSummary && (
                    <section className="admin-ops-note ai">
                        <div>
                            <strong>e仔自动回复</strong>
                            <span>
                                {aiSummary.enabled
                                    ? `${aiSummary.bot_ready ? 'Bot 已确认' : 'Bot 账号待确认'}，今日 ${compactNumber(aiSummary.today_used)} / ${compactNumber(aiSummary.daily_limit)}，待处理 ${compactNumber(aiSummary.pending)}，失败 ${compactNumber(aiSummary.failed)}`
                                    : '未启用，需要配置大模型 Key 和 e仔官方用户 ID。'}
                            </span>
                        </div>
                        <Link className="admin-button" to="/admin/assistant">
                            查看助手
                        </Link>
                    </section>
                )}

                <div className="admin-panel">
                    <div className="admin-panel-head">
                        <h2>热门内容</h2>
                        <Link to="/admin/posts">查看全部</Link>
                    </div>
                    <div className="admin-hot-list">
                        {hotPosts.length === 0 && <div className="admin-empty compact">还没有热门内容，先发几条官方攻略暖场。</div>}
                        {hotPosts.map((post, index) => (
                            <div className="admin-hot-item" key={post.id}>
                                <div className="admin-hot-rank">{index + 1}</div>
                                <div className="admin-hot-cover">
                                    {postCover(post) ? <img src={postCover(post)} alt="" /> : <FiStar />}
                                </div>
                                <div className="admin-hot-body">
                                    <strong>{post.title || '未命名内容'}</strong>
                                    <span>{postTypeText(post.post_type)} · {excerpt(post.content, 32)}</span>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </>}
        </div>
    );
};

export default AdminDashboard;
