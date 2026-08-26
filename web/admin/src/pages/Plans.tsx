import { ProTable, ModalForm, ProFormText, ProFormTextArea, ProFormDigit, ProFormSelect } from '@ant-design/pro-components';
import type { ActionType } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Space, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useRef } from 'react';
import { request, type Plan } from '../api/client';

// 可用渠道选项(与用户中心 Plans.tsx channelMap 一致, 写入 quota.channels)
const CHANNEL_OPTIONS = [
  { label: 'Webhook', value: 'webhook' },
  { label: '邮件', value: 'email' },
  { label: 'APNs', value: 'apns' },
  { label: 'FCM', value: 'fcm' },
  { label: '站内信', value: 'inapp' },
];

export default function Plans() {
  const actionRef = useRef<ActionType>();
  const isDefaultFree = (r: Plan) => r.price === 0;

  // 解析套餐额度 JSON, 取 monthly_messages, 解析失败返回 0
  const parseQuota = (r: Plan): number => {
    try {
      return JSON.parse(r.quota)?.monthly_messages || 0;
    } catch {
      return 0;
    }
  };

  // 解析套餐额度 JSON, 取 channels 数组, 解析失败返回空数组
  const parseQuotaChannels = (r: Plan): string[] => {
    try {
      const ch = JSON.parse(r.quota)?.channels;
      return Array.isArray(ch) ? ch : [];
    } catch {
      return [];
    }
  };

  return (
    <ProTable<Plan>
      headerTitle="套餐管理"
      rowKey="id"
      actionRef={actionRef}
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
              data: {
                ...values,
                quota: JSON.stringify({ monthly_messages: values.monthly_messages, channels: values.channels || [] }),
              },
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
          <ProFormSelect name="channels" label="可用渠道" mode="multiple" options={CHANNEL_OPTIONS} initialValue={['webhook']} />
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
        {
          title: '可用渠道',
          dataIndex: 'quota',
          width: 220,
          render: (_, r) => {
            const chs = parseQuotaChannels(r);
            if (chs.length === 0) return '-';
            return chs.map((c) => (
              <Tag key={c} style={{ marginBottom: 4 }}>
                {CHANNEL_OPTIONS.find((o) => o.value === c)?.label || c}
              </Tag>
            ));
          },
        },
        { title: '描述', dataIndex: 'description', ellipsis: true },
        {
          title: '操作',
          valueType: 'option',
          width: 140,
          render: (_, r) => (
            <Space size={8}>
              {/* 编辑: 所有套餐均显示, 表单预填当前值 */}
              <ModalForm<any>
                title="编辑套餐"
                trigger={<a>编辑</a>}
                initialValues={{
                  name: r.name,
                  price: r.price,
                  duration_days: r.duration_days,
                  monthly_messages: parseQuota(r),
                  channels: parseQuotaChannels(r),
                  description: r.description,
                }}
                onFinish={async (values) => {
                  await request({
                    url: `/admin/plans/${r.id}`,
                    method: 'PUT',
                    data: {
                      ...values,
                      quota: JSON.stringify({ monthly_messages: values.monthly_messages, channels: values.channels || [] }),
                    },
                  });
                  message.success('已保存');
                  actionRef.current?.reload();
                  return true;
                }}
                modalProps={{ destroyOnClose: true }}
              >
                <ProFormText name="name" label="套餐名称" rules={[{ required: true }]} />
                <ProFormDigit name="price" label="价格（元）" min={0} />
                <ProFormDigit name="duration_days" label="有效期（天）" min={1} />
                <ProFormDigit name="monthly_messages" label="月消息额度" min={0} />
                <ProFormSelect name="channels" label="可用渠道" mode="multiple" options={CHANNEL_OPTIONS} />
                <ProFormTextArea name="description" label="描述" />
              </ModalForm>
              {/* 删除: 免费版套餐不可删除 */}
              {!isDefaultFree(r) && (
                <Popconfirm title="删除该套餐？" onConfirm={async () => {
                  await request({ url: `/admin/plans/${r.id}`, method: 'DELETE' });
                  message.success('已删除');
                }}>
                  <a style={{ color: '#dc2626' }}>删除</a>
                </Popconfirm>
              )}
            </Space>
          ),
        },
      ]}
    />
  );
}
