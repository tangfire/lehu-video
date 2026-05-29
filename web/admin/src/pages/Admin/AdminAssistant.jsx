import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FiBookOpen, FiCpu, FiRefreshCw, FiShield, FiZap } from 'react-icons/fi';
import AdminAIReplies from './AdminAIReplies';
import AdminAuditSettings from './AdminAuditSettings';
import AdminKnowledge from './AdminKnowledge';
import './Admin.css';

const tabs = [
    { key: 'status', label: '回复状态', icon: <FiCpu /> },
    { key: 'knowledge', label: '知识库', icon: <FiBookOpen /> },
    { key: 'test', label: '测试提问', icon: <FiZap /> },
    { key: 'audit', label: '审核设置', icon: <FiShield /> },
    { key: 'failed', label: '失败任务', icon: <FiRefreshCw /> },
];

const AdminAssistant = () => {
    const [searchParams, setSearchParams] = useSearchParams();
    const initialTab = useMemo(() => {
        const tab = searchParams.get('tab');
        return tabs.some((item) => item.key === tab) ? tab : 'status';
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

    const knowledgeMode = activeTab === 'test' ? 'test' : 'documents';

    return (
        <div className="admin-merged-page assistant">
            <section className="admin-simple-head compact">
                <div>
                    <span className="admin-kicker">EZI ASSISTANT</span>
                    <h2>e仔助手</h2>
                    <p>日常只看状态和维护资料；失败任务、查询日志这类技术细节按需展开。</p>
                </div>
            </section>

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
                {activeTab === 'status' && <AdminAIReplies mode="summary" />}
                {activeTab === 'failed' && <AdminAIReplies mode="tasks" initialStatus="failed" />}
                {activeTab === 'audit' && <AdminAuditSettings />}
                {(activeTab === 'knowledge' || activeTab === 'test') && <AdminKnowledge mode={knowledgeMode} />}
            </div>
        </div>
    );
};

export default AdminAssistant;
