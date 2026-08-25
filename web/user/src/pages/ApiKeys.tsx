import { ProTable } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type ApiKey } from '../api/client';

export default function ApiKeys() {
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
      headerTitle="API Key 管理"
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
        { title: '状态', dataIndex: 'status', width: 90, render: (_, r) => <Tag color={r.status === 'active' ? 'success' : 'default'}>{r.status === 'active' ? '启用' : '禁用'}</Tag> },
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
