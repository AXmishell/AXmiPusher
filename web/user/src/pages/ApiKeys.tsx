import { ProTable, ModalForm, ProFormSelect, ProFormText, ProFormTextArea } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag, Tabs } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type ApiKey, type CompatKey } from '../api/client';

// ==================== Tab 1: 平台 API Key ====================

function ApiKeyTab() {
  const createKey = async () => {
    const d = await request<{ key: ApiKey; plain_key: string }>({
      url: '/api-keys',
      method: 'POST',
      data: { name: `key-${Date.now().toString().slice(-6)}`, scopes: '["message:send"]' },
    });
    message.success(`创建成功，请立即保存明文 Key: ${d.plain_key}`, 10);
    return true;
  };

  const revoke = async (id: number) => {
    await request({ url: `/api-keys/${id}`, method: 'DELETE' });
    message.success('已吊销');
    return true;
  };

  return (
    <ProTable<ApiKey>
      rowKey="id"
      search={false}
      toolBarRender={() => [
        <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => createKey().then(() => location.reload())}>
          新建 API Key
        </Button>,
      ]}
      request={async () => {
        const d = await request<{ data: ApiKey[]; total: number }>({ url: '/api-keys', method: 'GET' });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '名称', dataIndex: 'name', width: 180 },
        { title: 'Key 前缀', dataIndex: 'key_prefix', width: 140, render: (_, r) => <Tag>{r.key_prefix}...</Tag> },
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
  );
}

// ==================== Tab 2: 兼容 Key（Server酱） ====================

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
              { label: 'Server酱 v2 (sctapi)', value: 'serverchan_v2' },
              { label: 'Server酱 v1 (sc)', value: 'serverchan_v1' },
            ]}
          />
          <ProFormText
            name="external_key"
            label="外部 Key"
            placeholder="留空自动生成，可导入原 SendKey/SCKEY"
            tooltip="导入原 Server酱 的 SendKey，老脚本即可无缝切换"
          />
          <ProFormSelect name="default_channel" label="默认渠道" initialValue="webhook" options={[{ label: 'Webhook', value: 'webhook' }]} />
          <ProFormTextArea name="description" label="备注" />
        </ModalForm>,
      ]}
      request={async () => {
        const d = await request<{ data: CompatKey[]; total: number }>({ url: '/compat-keys', method: 'GET' });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        {
          title: '协议',
          dataIndex: 'source',
          width: 130,
          render: (_, r) => <Tag color={r.source === 'serverchan_v2' ? 'blue' : 'green'}>{r.source === 'serverchan_v2' ? 'Server酱 v2' : 'Server酱 v1'}</Tag>,
        },
        {
          title: '外部 Key',
          dataIndex: 'external_key',
          width: 200,
          copyable: true,
          render: (_, r) => (
            <code style={{ background: '#f1f5f9', padding: '2px 8px', borderRadius: 4, whiteSpace: 'nowrap' }}>{r.external_key}</code>
          ),
        },
        {
          title: '调用地址',
          width: 290,
          render: (_, r) => (
            <code style={{ fontSize: 12, color: '#1e40af', whiteSpace: 'nowrap' }}>
              POST /api/{r.source === 'serverchan_v2' ? 'sctapi' : 'sc'}/{r.external_key}.send
            </code>
          ),
        },
        { title: '备注', dataIndex: 'description', ellipsis: true },
        {
          title: '操作',
          valueType: 'option',
          width: 100,
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
        { key: 'api', label: 'API Key（服务端调用）', children: <ApiKeyTab /> },
        { key: 'compat', label: 'Server酱', children: <CompatKeyTab /> },
      ]}
    />
  );
}
