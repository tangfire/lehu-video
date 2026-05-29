import SparkMD5 from 'spark-md5';
import request from '../utils/request';

const IMAGE_TYPE_BY_MIME = {
    'image/jpeg': 'jpg',
    'image/png': 'png',
    'image/webp': 'webp',
};

const getImageFileType = (file) => {
    const byMime = IMAGE_TYPE_BY_MIME[file.type];
    if (byMime) return byMime;
    const match = /\.([a-zA-Z0-9]+)$/.exec(file.name || '');
    const ext = match ? match[1].toLowerCase() : '';
    if (ext === 'jpeg') return 'jpg';
    if (['jpg', 'png', 'webp'].includes(ext)) return ext;
    return '';
};

const hashFile = async (file) => {
    const buffer = await file.arrayBuffer();
    return SparkMD5.ArrayBuffer.hash(buffer);
};

const uploadPublicImage = async (file) => {
    const fileType = getImageFileType(file);
    if (!fileType) {
        throw new Error('仅支持 jpg、png、webp 图片');
    }
    const hash = await hashFile(file);
    const presign = await request.post('/campus/upload/presign', {
        media_type: 'image',
        hash,
        file_type: fileType,
        filename: file.name || `image.${fileType}`,
        size: file.size,
    });

    if (presign.upload_url) {
        const headers = {
            ...(presign.headers || {}),
        };
        if (!headers['Content-Type'] && !headers['content-type']) {
            headers['Content-Type'] = file.type || 'application/octet-stream';
        }
        const response = await fetch(presign.upload_url, {
            method: presign.method || 'PUT',
            headers,
            body: file,
        });
        if (!response.ok) {
            throw new Error(`图片直传失败（${response.status}）`);
        }
    }

    return request.post('/campus/upload/complete', {
        media_type: 'image',
        file_id: presign.file_id,
    });
};

export const campusAdminApi = {
    summary: () => request.get('/campus/admin/summary'),
    getAuditSettings: () => request.get('/campus/admin/settings/audit'),
    updateAuditSettings: (data) => request.put('/campus/admin/settings/audit', data),
    reconcileStats: () => request.post('/campus/admin/stats/reconcile'),
    listPosts: (params) => request.get('/campus/admin/posts', { params }),
    createPost: (data) => request.post('/campus/admin/posts', data),
    updatePost: (id, data) => request.put(`/campus/admin/posts/${id}`, data),
    deletePost: (id) => request.delete(`/campus/admin/posts/${id}`),
    batchPosts: (data) => request.post('/campus/admin/posts/batch', data),
    uploadImage: uploadPublicImage,
    listComments: (params) => request.get('/campus/admin/comments', { params }),
    deleteComment: (id) => request.delete(`/campus/admin/comments/${id}`),
    reviewComment: (id, data) => request.post(`/campus/moderation/comments/${id}/review`, data),
    aiReplySummary: () => request.get('/campus/admin/ai-replies/summary'),
    listAiReplyTasks: (params) => request.get('/campus/admin/ai-replies/tasks', { params }),
    retryAiReplyTask: (id) => request.post(`/campus/admin/ai-replies/tasks/${id}/retry`),
    listKnowledgeDocuments: (params) => request.get('/campus/admin/knowledge/documents', { params }),
    createKnowledgeDocument: (data) => request.post('/campus/admin/knowledge/documents', data),
    updateKnowledgeDocument: (id, data) => request.put(`/campus/admin/knowledge/documents/${id}`, data),
    reindexKnowledgeDocument: (id) => request.post(`/campus/admin/knowledge/documents/${id}/reindex`),
    listKnowledgeChunks: (id, params) => request.get(`/campus/admin/knowledge/documents/${id}/chunks`, { params }),
    testKnowledgeQuery: (data) => request.post('/campus/admin/knowledge/test-query', data),
    listKnowledgeQueryLogs: (params) => request.get('/campus/admin/knowledge/query-logs', { params }),
    uploadKnowledgeFile: (file) => {
        const formData = new FormData();
        formData.append('file', file);
        return request.post('/campus/admin/knowledge/upload', formData);
    },
    listReports: (params) => request.get('/campus/admin/reports', { params }),
    reviewReport: (id, data) => request.post(`/campus/admin/reports/${id}/review`, data),
    listFeedback: (params) => request.get('/campus/admin/feedback', { params }),
    reviewFeedback: (id, data) => request.post(`/campus/admin/feedback/${id}/review`, data),
    security: () => request.get('/campus/admin/security'),
    blockIP: (data) => request.post('/campus/admin/security/ip-blocks', data),
    unblockIP: (ip) => request.delete(`/campus/admin/security/ip-blocks/${encodeURIComponent(ip)}`),
    listUsers: (params) => request.get('/campus/admin/users', { params }),
    updateUserRole: (id, role) => request.put(`/campus/admin/users/${id}/role`, { role }),
    createNotification: (data) => request.post('/campus/admin/notifications', data),
    listCategories: () => request.get('/campus/forum/categories'),
};
