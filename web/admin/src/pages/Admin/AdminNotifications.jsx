import { useState } from 'react';
import { FiAlertTriangle, FiBell, FiSend } from 'react-icons/fi';
import { campusAdminApi } from '../../api/admin';
import './Admin.css';

const initialNotice = {
    title: '',
    content: '',
    link_page: 'community',
    link_params: '',
    audience: 'all_users',
};

const noticeTemplates = [
    {
        name: '内测欢迎',
        title: '欢迎体验深汕校园e站',
        content: '欢迎参加内测，可以先试试首页浏览、发帖、评论、收藏、消息中心和课表演示。遇到问题欢迎在“我的-意见反馈”告诉我们。',
        link_page: 'community',
    },
    {
        name: '维护提醒',
        title: '今晚将短暂维护',
        content: '今晚可能会进行一次短暂维护，期间部分页面可能偶发请求失败。维护完成后会恢复正常，感谢理解。',
        link_page: 'community',
    },
    {
        name: '内容征集',
        title: '来帮深汕新生补一条攻略',
        content: '如果你已经知道报到、宿舍、交通、校园网或社团相关信息，欢迎发一条笔记。优质内容会被深汕e仔精选到首页。',
        link_page: 'community',
    },
    {
        name: '课表测试',
        title: '课表功能内测提醒',
        content: '当前课表入口仍是演示导入链路，真实教务导入需要等学校确认开放。欢迎先体验页面流程，并反馈你希望看到的课表功能。',
        link_page: 'timetable',
    },
];

const AdminNotifications = () => {
    const [notice, setNotice] = useState(initialNotice);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [confirmText, setConfirmText] = useState('');

    const update = (key, value) => {
        setNotice((prev) => ({ ...prev, [key]: value }));
    };

    const applyTemplate = (template) => {
        setNotice((prev) => ({
            ...prev,
            ...template,
            audience: 'all_users',
            link_params: template.link_params || '',
        }));
        setMessage(`已填充通知模板：${template.name}`);
        setError('');
    };

    const validateNotice = () => {
        if (loading) return;
        if (!notice.title.trim() || !notice.content.trim()) {
            setError('请先填写通知标题和内容');
            return false;
        }
        if (notice.link_page === 'post-detail' && !String(notice.link_params || '').trim()) {
            setError('跳转到指定帖子时需要填写帖子 ID');
            return false;
        }
        return true;
    };

    const openConfirm = () => {
        if (!validateNotice()) return;
        setError('');
        setMessage('');
        setConfirmText('');
        setConfirmOpen(true);
    };

    const submit = async () => {
        if (!validateNotice()) return;
        if (notice.audience === 'all_users' && confirmText.trim() !== '发送') {
            setError('请输入“发送”确认本次全量通知');
            return;
        }
        setLoading(true);
        setMessage('');
        setError('');
        try {
            const res = await campusAdminApi.createNotification({
                title: notice.title.trim(),
                content: notice.content.trim(),
                link_page: notice.link_page || 'community',
                link_params: parseNoticeParams(notice),
                audience: notice.audience || 'all_users',
            });
            const taskId = res?.task_id ? `，任务ID：${res.task_id}` : '';
            setMessage(`系统通知已加入可靠发送任务${taskId}`);
            setNotice(initialNotice);
            setConfirmOpen(false);
            setConfirmText('');
        } catch (err) {
            setError(err.message || '系统通知发送失败');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="admin-compose-page">
            {message && <div className="admin-toast success">{message}</div>}
            {error && <div className="admin-error">{error}</div>}
            <div className="admin-composer-grid simple">
                <section className="admin-panel">
                    <div className="admin-panel-head">
                        <div>
                            <h2>发布系统通知</h2>
                            <p>用于内测公告、维护提醒、活动提醒，会出现在小程序“我的 - 消息中心”。</p>
                        </div>
                        <button className="admin-button primary" type="button" disabled={loading} onClick={openConfirm}>
                            <FiSend />
                            发送给全部用户
                        </button>
                    </div>

                    <div className="admin-step-title">1. 选择通知场景</div>
                    <div className="admin-template-grid">
                        {noticeTemplates.map((template) => (
                            <button className="admin-template-card" type="button" key={template.name} onClick={() => applyTemplate(template)}>
                                <span>通知</span>
                                <strong>{template.name}</strong>
                                <p>{template.title}</p>
                            </button>
                        ))}
                    </div>

                    <div className="admin-step-title">2. 编辑通知内容</div>
                    <div className="admin-form simple-compose">
                        <div className="admin-field wide">
                            <label>通知标题</label>
                            <input className="admin-input" value={notice.title} onChange={(e) => update('title', e.target.value)} maxLength={60} placeholder="例如：今晚 10 点短暂维护" />
                        </div>
                        <div className="admin-field wide">
                            <label>通知内容</label>
                            <textarea className="admin-textarea tall" value={notice.content} onChange={(e) => update('content', e.target.value)} maxLength={500} placeholder="写清楚对用户有什么影响，尽量一句话能看懂。" />
                        </div>
                        <div className="admin-field">
                            <label>点击后打开</label>
                            <select className="admin-select" value={notice.link_page} onChange={(e) => update('link_page', e.target.value)}>
                                <option value="community">首页</option>
                                <option value="timetable">课表</option>
                                <option value="post-detail">指定帖子</option>
                            </select>
                        </div>
                        {notice.link_page === 'post-detail' && (
                            <div className="admin-field">
                                <label>帖子 ID</label>
                                <input className="admin-input" value={notice.link_params} onChange={(e) => update('link_params', e.target.value)} placeholder="填写帖子 ID" />
                            </div>
                        )}
                    </div>
                </section>

                <aside className="admin-compose-aside">
                    <section className="admin-panel admin-notice-panel">
                        <h2>发送规则</h2>
                        <p className="admin-muted">当前 v1 只支持发送给全部用户，适合内测规模。后续用户量变大后，再增加分组发送和历史通知列表。</p>
                        <div className="admin-simple-switch-row">
                            <button className="admin-pill active" type="button">全部用户</button>
                            <button className="admin-pill" type="button" disabled>分组发送</button>
                        </div>
                    </section>

                    <section className="admin-preview-panel">
                        <div className="admin-phone-preview">
                            <div className="admin-preview-cover">
                                <div><FiBell /> 系统通知</div>
                            </div>
                            <div className="admin-preview-body">
                                <h3>{notice.title || '通知标题会显示在消息中心'}</h3>
                                <p>{notice.content || '通知内容建议短一点，告诉用户发生了什么、需要做什么。'}</p>
                                <div className="admin-preview-author">
                                    <span>深汕校园e站</span>
                                    <em>系统</em>
                                </div>
                                <div className="admin-muted">点击打开：{linkPageText(notice.link_page)}</div>
                            </div>
                        </div>
                    </section>
                </aside>
            </div>

            {confirmOpen && (
                <div className="admin-modal-backdrop" role="presentation">
                    <div className="admin-confirm-modal">
                        <div className="admin-modal-icon danger"><FiAlertTriangle /></div>
                        <h3>确认发送给全部用户？</h3>
                        <p>这条通知会进入所有用户的小程序消息中心。发送任务创建后不能一键撤回，请确认标题、正文和跳转入口无误。</p>
                        <div className="admin-confirm-preview">
                            <strong>{notice.title}</strong>
                            <span>{notice.content}</span>
                            <em>点击打开：{linkPageText(notice.link_page)}</em>
                        </div>
                        <label className="admin-confirm-input">
                            输入“发送”确认
                            <input className="admin-input" value={confirmText} onChange={(e) => setConfirmText(e.target.value)} placeholder="发送" autoFocus />
                        </label>
                        <div className="admin-modal-actions">
                            <button className="admin-button" disabled={loading} onClick={() => setConfirmOpen(false)}>取消</button>
                            <button className="admin-button danger" disabled={loading || confirmText.trim() !== '发送'} onClick={submit}>
                                {loading ? '加入任务中...' : '确认发送'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

function parseNoticeParams(notice) {
    if (notice.link_page !== 'post-detail') return {};
    const id = String(notice.link_params || '').trim();
    return id ? { id } : {};
}

function linkPageText(page) {
    if (page === 'timetable') return '课表';
    if (page === 'post-detail') return '指定帖子';
    return '首页';
}

export default AdminNotifications;
