import { useState } from 'react';
import { ProForm, ProFormText, ProFormTextArea, ProFormSelect } from '@ant-design/pro-components';
import { Card, Alert, Typography, message } from 'antd';
import { request } from '../api/client';

export default function SendMessage() {
  const [result, setResult] = useState<any>(null);

  const onFinish = async (values: any) => {
    try {
      const d = await request<{ message_id: number; duplicate: boolean; count: number }>({
        url: '/messages',
        method: 'POST',
        data: {
          request_id: values.request_id || undefined,
          title: values.title,
          content: values.content,
          channel: values.channel || 'webhook',
          recipients: [{ target: values.target, params: {} }],
        },
      });
      setResult(d);
      message.success(`发送受理成功 message_id=${d.message_id}`);
      return true;
    } catch (e: any) {
      message.error(e.message || '发送失败');
      return false;
    }
  };

  return (
    <Card>
      <Typography.Title level={4} style={{ marginTop: 0 }}>发送消息</Typography.Title>
      <ProForm
        onFinish={onFinish}
        initialValues={{ channel: 'webhook', target: 'webhook-endpoint' }}
        submitter={{ searchConfig: { submitText: '发送', resetText: '重置' } }}
      >
        <ProFormText name="request_id" label="幂等键 request_id" placeholder="留空则每次都是新消息" tooltip="同一 request_id 重复发送只受理一次" />
        <ProFormText name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]} placeholder="消息标题" />
        <ProFormTextArea name="content" label="内容" rules={[{ required: true, message: '请输入内容' }]} placeholder="消息正文" fieldProps={{ rows: 4 }} />
        <ProFormSelect
          name="channel"
          label="渠道"
          options={[
            { label: 'Webhook（回调地址）', value: 'webhook' },
            { label: '自动路由', value: 'auto' },
          ]}
        />
        <ProFormText name="target" label="收件目标" rules={[{ required: true, message: '请输入收件目标' }]} placeholder="Webhook 场景可为任意标识" />
        {result && (
          <Alert
            style={{ marginBottom: 16 }}
            type={result.duplicate ? 'warning' : 'success'}
            showIcon
            message={`受理成功: message_id=${result.message_id}`}
            description={result.duplicate ? '注意: 该 request_id 已受理过, 返回的是原消息。' : `本批共 ${result.count} 条消息已进入队列。`}
          />
        )}
      </ProForm>
    </Card>
  );
}
