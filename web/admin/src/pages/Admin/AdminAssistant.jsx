import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FiBookOpen, FiCpu, FiRefreshCw, FiShield, FiSliders, FiZap } from 'react-icons/fi';
import AdminAIReplies from './AdminAIReplies';
import AdminAuditSettings from './AdminAuditSettings';
import AdminEzaiPersona from './AdminEzaiPersona';
import AdminKnowledge from './AdminKnowledge';
import './Admin.css';

const tabs = [
    { key: 'status', label: '回复状态', icon: <FiCpu /> },
    { key: 'persona', label: '人设设定', icon: <FiSliders /> },
    { key: 'knowledge', label: '知识库', icon: <FiBookOpen /> },
    { key: 'test', label: '知识库测试', icon: <FiZap /> },
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
                {activeTab === 'status' && <AdminAIReplies mode="tasks" initialStatus="" />}
                {activeTab === 'persona' && <AdminEzaiPersona />}
                {activeTab === 'failed' && <AdminAIReplies mode="tasks" initialStatus="failed" />}
                {activeTab === 'audit' && <AdminAuditSettings />}
                {(activeTab === 'knowledge' || activeTab === 'test') && <AdminKnowledge mode={knowledgeMode} />}
            </div>
        </div>
    );
};

export default AdminAssistant;
