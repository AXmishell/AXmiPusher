import { useState, useRef } from 'react';
import { ProTable, ModalForm, ProFormSelect, ProFormText, ProFormTextArea, type ActionType } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag, Tabs, Modal, Alert } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type ApiKey, type CompatKey } from '../api/client';

// ==================== Tab 1: 平台 API Key ====================

function ApiKeyTab() {
  const actionRef = useRef<ActionType>();
  // 明文 Key 仅创建时展示一次(后端只存哈希), 用弹窗承载并支持点击复制
  const [plainKey, setPlainKey] = useState<string | null>(null);

  const createKey = async () => {
    const d = await request<{ key: ApiKey; plain_key: string }>({
      url: '/api-keys',
      method: 'POST',
      data: { name: `key-${Date.now().toString().slice(-6)}`, scopes: '["message:send"]' },
    });
    setPlainKey(d.plain_key);
    return true;
  };

  const revoke = async (id: number) => {
    await request({ url: `/api-keys/${id}`, method: 'DELETE' });
    message.success('已吊销');
    return true;
  };

  return (
    <>
      <ProTable<ApiKey>
        rowKey="id"
        actionRef={actionRef}
        search={false}
        toolBarRender={() => [
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => createKey().then(() => actionRef.current?.reload())}>
            新建兼容API Key
          </Button>,
        ]}
        request={async () => {
          const d = await request<{ data: ApiKey[]; total: number }>({ url: '/api-keys', method: 'GET' });
          return { data: d.data, total: d.total, success: true };
        }}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: '名称', dataIndex: 'name', width: 180 },
          {
            title: 'Key 前缀',
            dataIndex: 'key_prefix',
            ellipsis: true,
            render: (_, r) => (
              <Tag style={{ cursor: 'pointer' }} title="点击复制前缀" onClick={() => copyText(r.key_prefix)}>
                {r.key_prefix}...
              </Tag>
            ),
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 90,
            render: (_, r) => <Tag color={r.status === 'active' ? 'success' : 'default'}>{r.status === 'active' ? '启用' : '禁用'}</Tag>,
          },
          { title: '最后使用', dataIndex: 'last_used_at', valueType: 'dateTime', width: 170 },
          {
            title: '操作',
            valueType: 'option',
            width: 100,
            render: (_, r) => [
              <Popconfirm key="del" title="确定吊销该 Key？" onConfirm={() => revoke(r.id).then(() => location.reload())}>
                <a style={{ color: '#dc2626' }}>吊销</a>
              </Popconfirm>,
            ],
          },
        ]}
      />
      {/* 创建成功: 明文 Key 一次性展示, 点击复制 */}
      <Modal
        title="创建成功 - 请立即保存明文 Key"
        open={!!plainKey}
        onCancel={() => { setPlainKey(null); actionRef.current?.reload(); }}
        footer={[
          <Button key="copy" type="primary" onClick={() => plainKey && copyText(plainKey)}>复制 Key</Button>,
          <Button key="done" onClick={() => { setPlainKey(null); actionRef.current?.reload(); }}>我已保存</Button>,
        ]}
      >
        <Alert type="warning" showIcon message="明文 Key 仅本次创建时展示, 关闭后无法再次查看, 请立即复制保存" style={{ marginBottom: 12 }} />
        <div
          onClick={() => plainKey && copyText(plainKey)}
          title="点击复制"
          style={{
            background: '#f1f5f9',
            padding: '10px 12px',
            borderRadius: 6,
            cursor: 'pointer',
            userSelect: 'all',
            fontFamily: 'monospace',
            wordBreak: 'break-all',
          }}
        >
          {plainKey}
        </div>
      </Modal>
    </>
  );
}

// ==================== Tab 2: 兼容 Key（Server酱） ====================

// 兼容 Key 可选的默认渠道(与发送消息页渠道一致, 后端按该值透传为消息 channel)
const COMPAT_CHANNEL_OPTIONS = [
  { label: 'Webhook（回调地址）', value: 'webhook' },
  { label: '邮件（SMTP）', value: 'email' },
  { label: 'APNs（iOS 推送）', value: 'apns' },
  { label: 'FCM（Android 推送）', value: 'fcm' },
  { label: '站内信', value: 'inapp' },
  { label: '自动路由', value: 'auto' },
];

// 复制文本到剪贴板: 优先 navigator.clipboard(需安全上下文 https/localhost),
// 否则降级 textarea + execCommand(兼容 http 生产环境)。
const copyText = async (text: string) => {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
    } else {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
    }
    message.success('已复制到剪贴板');
  } catch {
    message.error('复制失败，请手动选择复制');
  }
};

function CompatKeyTab() {
  return (
    <ProTable<CompatKey>
      rowKey="id"
      search={false}
      toolBarRender={() => [
        <ModalForm<{ source: string; external_key?: string; default_channel?: string; description?: string }>
          key="new"
          title="新增 Key"
          trigger={<Button type="primary" icon={<PlusOutlined />}>新增 Key</Button>}
          onFinish={async (values) => {
            await request({ url: '/compat-keys', method: 'POST', data: values });
            message.success('创建成功');
            return true;
          }}
          modalProps={{ destroyOnClose: true }}
        >
          <ProFormSelect
            name="source"
            label="兼容协议"
            rules={[{ required: true }]}
            options={[
              { label: 'Server酱·Turbo版 (sctapi)', value: 'serverchan_v2' },
              { label: 'Server酱3 (sc)', value: 'serverchan_v1' },
            ]}
          />
          <ProFormText
            name="external_key"
            label="外部 Key"
            placeholder="留空自动生成，可导入原 SendKey/SCKEY"
            tooltip="导入原 Server酱 的 Key，老脚本即可无缝切换"
          />
          <ProFormSelect name="default_channel" label="默认渠道" initialValue="webhook" options={COMPAT_CHANNEL_OPTIONS} />
          <ProFormTextArea name="description" label="备注" />
        </ModalForm>,
      ]}
      request={async () => {
        const d = await request<{ data: CompatKey[]; total: number }>({ url: '/compat-keys', method: 'GET' });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 60 },
        {
          title: '协议',
          dataIndex: 'source',
          width: 120,
          render: (_, r) => <Tag color={r.source === 'serverchan_v2' ? 'blue' : 'green'}>{r.source === 'serverchan_v2' ? 'Server酱·Turbo版' : 'Server酱3'}</Tag>,
        },
        {
          title: '外部 Key',
          dataIndex: 'external_key',
          // 自适应列宽 + 左对齐; 点击整段 Key 复制到剪贴板(解决原固定宽列过长无法完整复制的问题)
          ellipsis: true,
          render: (_, r) => (
            <code
              title="点击复制"
              onClick={() => copyText(r.external_key)}
              style={{
                background: '#f1f5f9',
                padding: '2px 8px',
                borderRadius: 4,
                cursor: 'pointer',
                userSelect: 'all',
                whiteSpace: 'nowrap',
              }}
            >
              {r.external_key}
            </code>
          ),
        },
        {
          title: '调用地址',
          // 自适应列宽 + 左对齐; 地址按当前访问网址自动拼接完整 URL;
          // 显示带方法(GET=sc / POST=sctapi), 溢出以省略号结尾; 点击复制仅复制完整 URL(不含方法)
          ellipsis: true,
          render: (_, r) => {
            const method = r.source === 'serverchan_v2' ? 'POST' : 'GET';
            const url = `${window.location.origin}/api/${r.source === 'serverchan_v2' ? 'sctapi' : 'sc'}/${r.external_key}.send`;
            return (
              <code
                title={`${method} ${url}（点击复制完整链接）`}
                onClick={() => copyText(url)}
                style={{
                  fontSize: 12,
                  color: '#1e40af',
                  cursor: 'pointer',
                  whiteSpace: 'nowrap',
                  display: 'inline-block',
                  maxWidth: '100%',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  verticalAlign: 'bottom',
                }}
              >
                {method} {url}
              </code>
            );
          },
        },
        { title: '备注', dataIndex: 'description', ellipsis: true },
        {
          title: '操作',
          valueType: 'option',
          width: 80,
          render: (_, r) => [
            <Popconfirm
              key="del"
              title="确定删除？删除后该 Key 立即失效"
              onConfirm={async () => {
                await request({ url: `/compat-keys/${r.id}`, method: 'DELETE' });
                message.success('已删除');
              }}
            >
              <a style={{ color: '#dc2626' }}>删除</a>
            </Popconfirm>,
          ],
        },
      ]}
    />
  );
}

// ==================== 合并页面 ====================

export default function ApiKeys() {
  return (
    <Tabs
      defaultActiveKey="api"
      items={[
        { key: 'api', label: '兼容API Key（服务端调用）', children: <ApiKeyTab /> },
        { key: 'compat', label: 'Server酱', children: <CompatKeyTab /> },
      ]}
    />
  );
}
