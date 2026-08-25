import { ProTable, ProForm, ModalForm, ProFormSelect, ProFormText, ProFormTextArea } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type CompatKey } from '../api/client';

export default function CompatKeys() {
  return (
    <ProTable<CompatKey>
      headerTitle="兼容 Key（Server酱）"
      rowKey="id"
      search={false}
      toolBarRender={() => [
        <ModalForm<{ source: string; external_key?: string; default_channel?: string; description?: string }>
          key="new"
          title="新建兼容 Key"
          trigger={<Button type="primary" icon={<PlusOutlined />}>新建兼容 Key</Button>}
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
          <ProFormText name="external_key" label="外部 Key" placeholder="留空自动生成，可导入原 SendKey/SCKEY" tooltip="导入原 Server酱 的 SendKey，老脚本即可无缝切换" />
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
          width: 140,
          render: (_, r) => <Tag color={r.source === 'serverchan_v2' ? 'blue' : 'green'}>{r.source === 'serverchan_v2' ? 'Server酱 v2' : 'Server酱 v1'}</Tag>,
        },
        {
          title: '外部 Key',
          dataIndex: 'external_key',
          width: 220,
          copyable: true,
          render: (_, r) => (
            <code style={{ background: '#f1f5f9', padding: '2px 8px', borderRadius: 4, whiteSpace: 'nowrap' }}>{r.external_key}</code>
          ),
        },
        {
          title: '调用地址',
          width: 300,
          render: (_, r) => (
            <code style={{ fontSize: 12, color: '#1e40af', whiteSpace: 'nowrap' }}>
              POST /api/{r.source === 'serverchan_v2' ? 'sctapi' : 'sc'}/{r.external_key}.send
            </code>
          ),
        },
        { title: '备注', dataIndex: 'description', ellipsis: true },
        { title: '最后使用', dataIndex: 'last_used_at', valueType: 'dateTime', width: 170 },
        {
          title: '操作',
          valueType: 'option',
          width: 100,
          render: (_, r) => [
            <Popconfirm key="del" title="确定删除？删除后该 Key 立即失效" onConfirm={async () => {
              await request({ url: `/compat-keys/${r.id}`, method: 'DELETE' });
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
