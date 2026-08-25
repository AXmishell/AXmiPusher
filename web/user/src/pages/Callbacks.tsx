import { ProTable, ModalForm, ProFormText, ProFormTextArea, ProFormSelect } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type Callback } from '../api/client';

export default function Callbacks() {
  return (
    <ProTable<Callback>
      headerTitle="回调订阅（Webhook 送达地址）"
      rowKey="id"
      search={false}
      toolBarRender={() => [
        <ModalForm<any>
          key="new"
          title="注册回调地址"
          trigger={<Button type="primary" icon={<PlusOutlined />}>注册回调</Button>}
          onFinish={async (values) => {
            await request({
              url: '/callbacks',
              method: 'POST',
              data: { url: values.url, secret: values.secret, events: values.events?.split(',') ?? ['success', 'failed'] },
            });
            message.success('注册成功');
            return true;
          }}
          modalProps={{ destroyOnClose: true }}
        >
          <ProFormText name="url" label="回调 URL" rules={[{ required: true, type: 'url', message: '请输入有效 URL' }]} placeholder="https://your-server.com/hook" />
          <ProFormText name="secret" label="签名密钥" placeholder="用于 HMAC 签名校验，留空则不签名" tooltip="回调带 X-MP-Signature 头 (HMAC-SHA256)" />
          <ProFormText name="events" label="订阅事件（逗号分隔）" initialValue="success,failed" tooltip="success,failed,retry,dead" />
        </ModalForm>,
      ]}
      request={async () => {
        const d = await request<{ data: Callback[]; total: number }>({ url: '/callbacks', method: 'GET' });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '回调 URL', dataIndex: 'url', ellipsis: true },
        { title: '订阅事件', dataIndex: 'events', width: 200, render: (_, r) => (r.events || '').split(',').filter(Boolean).map((e) => <Tag key={e}>{e}</Tag>) },
        { title: '状态', dataIndex: 'status', width: 90, render: (_, r) => <Tag color={r.status === 'active' ? 'success' : 'default'}>{r.status === 'active' ? '启用' : '禁用'}</Tag> },
        {
          title: '操作',
          valueType: 'option',
          width: 100,
          render: (_, r) => [
            <Popconfirm key="del" title="确定删除？" onConfirm={async () => {
              await request({ url: `/callbacks/${r.id}`, method: 'DELETE' });
              message.success('已删除');
            }}>
              <a style={{ color: '#dc2626' }}>删除</a>
            </Popconfirm>,
          ],
        },
      ]}
    />
  );
}
