import { ProTable, ModalForm, ProFormText, ProFormTextArea, ProFormSelect } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag, Progress, Typography } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { request } from '../api/client';

interface BatchTask {
  id: number;
  name: string;
  status: string; // pending|running|done|failed|cancelled
  total: number;
  success: number;
  failed: number;
  error: string;
  created_at: string;
}

const statusMap: Record<string, { color: string; text: string }> = {
  pending: { color: 'default', text: '等待中' },
  running: { color: 'processing', text: '执行中' },
  done: { color: 'success', text: '完成' },
  failed: { color: 'error', text: '失败' },
  cancelled: { color: 'warning', text: '已取消' },
};

export default function BatchTasks() {
  const [refreshKey, setRefreshKey] = useState(0);

  return (
    <ProTable<BatchTask>
      headerTitle="批量任务"
      rowKey="id"
      key={refreshKey}
      search={false}
      toolBarRender={() => [
        <ModalForm<any>
          key="new"
          title="创建批量任务"
          width={560}
          trigger={<Button type="primary" icon={<PlusOutlined />}>创建批量任务</Button>}
          onFinish={async (values) => {
            const targets = (values.targets || '')
              .split('\n')
              .map((s: string) => s.trim())
              .filter(Boolean);
            if (targets.length === 0) {
              message.error('收件人列表不能为空');
              return false;
            }
            const recipients = targets.map((t: string) => ({ target: t, params: {} }));
            await request({
              url: '/batch-tasks',
              method: 'POST',
              data: {
                name: values.name,
                template_code: values.template_code || undefined,
                content: values.content,
                channel: values.channel || 'webhook',
                recipients,
              },
            });
            message.success(`任务已创建（${recipients.length} 个收件人）`);
            setRefreshKey((k) => k + 1);
            return true;
          }}
          modalProps={{ destroyOnClose: true }}
        >
          <ProFormText name="name" label="任务名称" rules={[{ required: true }]} />
          <ProFormSelect
            name="channel"
            label="渠道"
            initialValue="webhook"
            options={[
              { label: 'Webhook', value: 'webhook' },
              { label: '邮件', value: 'email' },
              { label: '站内信', value: 'inapp' },
            ]}
          />
          <ProFormText name="template_code" label="模板编码" tooltip="留空则使用下方内容" />
          <ProFormTextArea name="content" label="内容（无模板时）" fieldProps={{ rows: 2 }} />
          <ProFormTextArea
            name="targets"
            label="收件人列表（每行一个）"
            rules={[{ required: true }]}
            fieldProps={{ rows: 6 }}
            placeholder={'user1@example.com\nuser2@example.com\n...'}
          />
        </ModalForm>,
      ]}
      request={async (params) => {
        const { current = 1, pageSize = 20 } = params as any;
        const d = await request<{ data: BatchTask[]; total: number }>({
          url: '/batch-tasks',
          method: 'GET',
          params: { current, pageSize },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '名称', dataIndex: 'name', width: 160 },
        {
          title: '状态',
          dataIndex: 'status',
          width: 100,
          render: (_, r) => {
            const s = statusMap[r.status] ?? { color: 'default', text: r.status };
            return <Tag color={s.color}>{s.text}</Tag>;
          },
        },
        {
          title: '进度',
          dataIndex: 'progress',
          render: (_, r) => {
            const pct = r.total > 0 ? Math.round(((r.success + r.failed) / r.total) * 100) : 0;
            return (
              <div style={{ width: 160 }}>
                <Progress
                  percent={pct}
                  size="small"
                  status={r.status === 'failed' ? 'exception' : undefined}
                  format={() => `${r.success + r.failed}/${r.total}`}
                />
              </div>
            );
          },
        },
        { title: '成功', dataIndex: 'success', width: 80, render: (_, r) => <Typography.Text type="success">{r.success}</Typography.Text> },
        { title: '失败', dataIndex: 'failed', width: 80, render: (_, r) => <Typography.Text type="danger">{r.failed}</Typography.Text> },
        { title: '错误', dataIndex: 'error', ellipsis: true },
        { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
        {
          title: '操作',
          valueType: 'option',
          width: 90,
          render: (_, r) =>
            r.status === 'pending' || r.status === 'running'
              ? [
                  <Popconfirm key="c" title="取消后停止后续发送，确定？" onConfirm={async () => {
                    await request({ url: `/batch-tasks/${r.id}/cancel`, method: 'POST', data: {} });
                    message.success('已取消');
                    setRefreshKey((k) => k + 1);
                  }}>
                    <a style={{ color: '#dc2626' }}>取消</a>
                  </Popconfirm>,
                ]
              : [],
        },
      ]}
      pagination={{ defaultPageSize: 10 }}
    />
  );
}
