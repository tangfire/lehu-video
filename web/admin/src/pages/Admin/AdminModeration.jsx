import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FiFlag, FiMessageCircle, FiSend } from 'react-icons/fi';
import AdminComments from './AdminComments';
import AdminFeedback from './AdminFeedback';
import AdminReports from './AdminReports';
import './Admin.css';

const tabs = [
    { key: 'reports', label: '举报处理', icon: <FiFlag /> },
    { key: 'feedback', label: '用户反馈', icon: <FiSend /> },
    { key: 'comments', label: '评论管理', icon: <FiMessageCircle /> },
];

const AdminModeration = () => {
    const [searchParams, setSearchParams] = useSearchParams();
    const initialTab = useMemo(() => {
        const tab = searchParams.get('tab');
        return tabs.some((item) => item.key === tab) ? tab : 'reports';
    }, [searchParams]);
    const [activeTab, setActiveTab] = useState(initialTab);

    useEffect(() => {
        setActiveTab(initialTab);
    }, [initialTab]);

    const switchTab = (key) => {
        setActiveTab(key);
        const next = new URLSearchParams(searchParams);
        next.set('tab', key);
        setSearchParams(next, { replace: true });
    };

    return (
        <div className="admin-merged-page">
            <div className="admin-page-tabs">
                {tabs.map((item) => (
                    <button
                        className={activeTab === item.key ? 'active' : ''}
                        type="button"
                        key={item.key}
                        onClick={() => switchTab(item.key)}
                    >
                        {item.icon}
                        {item.label}
                    </button>
                ))}
            </div>

            <div className="admin-tab-panel">
                {activeTab === 'reports' && <AdminReports />}
                {activeTab === 'feedback' && <AdminFeedback />}
                {activeTab === 'comments' && <AdminComments />}
            </div>
        </div>
    );
};

export default AdminModeration;
