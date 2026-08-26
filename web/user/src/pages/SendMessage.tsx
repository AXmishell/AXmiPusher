import { useState, useRef } from 'react';
import { ProForm, ProFormText, ProFormTextArea, ProFormSelect, type ProFormInstance } from '@ant-design/pro-components';
import { Card, Alert, Typography, message } from 'antd';
import { request } from '../api/client';

const CHANNEL_OPTIONS = [
  { label: 'Webhook（回调地址）', value: 'webhook' },
  { label: '邮件（SMTP）', value: 'email' },
  { label: 'APNs（iOS 推送）', value: 'apns' },
  { label: 'FCM（Android 推送）', value: 'fcm' },
  { label: '站内信', value: 'inapp' },
  { label: '自动路由', value: 'auto' },
];

// 各渠道收件目标默认值: 切换渠道时同步重置, 防止残留上一渠道的值导致发送死信。
const CHANNEL_TARGET_DEFAULTS: Record<string, string> = {
  webhook: 'webhook-endpoint',
  inapp: 'all', // 站内信默认发给全部用户
  email: '', // 邮件留空 = 使用通道配置的默认收件人
  apns: '',
  fcm: '',
  auto: '',
};

export default function SendMessage() {
  const [result, setResult] = useState<any>(null);
  const [channel, setChannel] = useState('webhook');
  const formRef = useRef<ProFormInstance>(null);

  // 收件目标字段按渠道动态适配(label / required / placeholder)
  const targetMeta = (() => {
    switch (channel) {
      case 'email':
        return { label: '收件人邮箱', required: false, placeholder: '留空使用通道配置的默认收件人' };
      case 'apns':
      case 'fcm':
        return { label: '设备 Token', required: true, placeholder: '设备 token' };
      case 'inapp':
        return { label: '收件人', required: false, placeholder: '邮箱或 all(默认 all, 发给全部用户)' };
      case 'auto':
        return { label: '收件目标', required: true, placeholder: '目标标识' };
      default:
        return { label: '收件目标', required: true, placeholder: '回调地址/标识' }; // webhook
    }
  })();

  const onFinish = async (values: any) => {
    const ch = values.channel || 'webhook';
    // email 留空 → 后端使用通道配置的默认收件人; inapp 留空 → 发给全部用户
    let target: string | undefined = values.target;
    if (ch === 'email') {
      target = values.target || undefined;
    } else if (ch === 'inapp') {
      target = values.target || 'all';
    }
    try {
      const d = await request<{ message_id: number; duplicate: boolean; count: number }>({
        url: '/messages',
        method: 'POST',
        data: {
          request_id: values.request_id || undefined,
          title: values.title,
          content: values.content,
          channel: ch,
          recipients: [{ target, params: {} }],
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
        formRef={formRef}
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
          options={CHANNEL_OPTIONS}
          fieldProps={{
            onChange: (v: any) => {
              setChannel(v);
              // 切换渠道时重置收件目标为对应默认值, 避免残留上一渠道的值(如 webhook 默认值)导致死信。
              formRef.current?.setFieldsValue({ target: CHANNEL_TARGET_DEFAULTS[v] ?? '' });
            },
          }}
        />
        <ProFormText
          name="target"
          label={targetMeta.label}
          rules={targetMeta.required ? [{ required: true, message: '请输入收件目标' }] : []}
          placeholder={targetMeta.placeholder}
        />
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
