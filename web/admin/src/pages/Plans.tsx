import { ProTable, ModalForm, ProFormText, ProFormTextArea, ProFormDigit, ProFormSelect } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type Plan } from '../api/client';

export default function Plans() {
  const isDefaultFree = (r: Plan) => r.price === 0;

  return (
    <ProTable<Plan>
      headerTitle="套餐管理"
      rowKey="id"
      search={false}
      toolBarRender={() => [
        <ModalForm<any>
          key="new"
          title="新建套餐"
          trigger={<Button type="primary" icon={<PlusOutlined />}>新建套餐</Button>}
          onFinish={async (values) => {
            await request({
              url: '/admin/plans',
              method: 'POST',
              data: { ...values, quota: JSON.stringify({ monthly_messages: values.monthly_messages }) },
            });
            message.success('创建成功');
            return true;
          }}
          modalProps={{ destroyOnClose: true }}
        >
          <ProFormText name="name" label="套餐名称" rules={[{ required: true }]} />
          <ProFormDigit name="price" label="价格（元）" initialValue={0} min={0} />
          <ProFormDigit name="duration_days" label="有效期（天）" initialValue={30} min={1} />
          <ProFormDigit name="monthly_messages" label="月消息额度" initialValue={10000} min={0} />
          <ProFormTextArea name="description" label="描述" />
        </ModalForm>,
      ]}
      request={async () => {
        const d = await request<{ data: Plan[]; total: number }>({ url: '/admin/plans', method: 'GET' });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '名称', dataIndex: 'name', width: 140 },
        {
          title: '价格',
          dataIndex: 'price',
          width: 110,
          render: (_, r) => (r.price === 0 ? <Tag color="green">免费</Tag> : `¥${r.price}`),
        },
        { title: '有效期', dataIndex: 'duration_days', width: 100, render: (_, r) => `${r.duration_days} 天` },
        {
          title: '额度',
          dataIndex: 'quota',
          width: 150,
          render: (_, r) => {
            try {
              const q = JSON.parse(r.quota);
              return <Tag color="blue">{Number(q.monthly_messages).toLocaleString()} 条/月</Tag>;
            } catch {
              return '-';
            }
          },
        },
        { title: '描述', dataIndex: 'description', ellipsis: true },
        {
          title: '操作',
          valueType: 'option',
          width: 100,
          render: (_, r) =>
            isDefaultFree(r)
              ? []
              : [
                  <Popconfirm key="del" title="删除该套餐？" onConfirm={async () => {
                    await request({ url: `/admin/plans/${r.id}`, method: 'DELETE' });
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
