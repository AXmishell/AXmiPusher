import { useState } from 'react';
import { Modal, Form, Input, message } from 'antd';
import { request } from '../api/client';

interface Props {
  open: boolean;
  onClose: () => void;
}

// 修改密码弹窗: 旧密码校验 + 新密码(≥8位) + 确认一致。
export default function ChangePasswordModal({ open, onClose }: Props) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const onFinish = async (values: { old: string; new: string }) => {
    setLoading(true);
    try {
      await request({ url: '/admin/auth/change-password', method: 'POST', data: { old_password: values.old, new_password: values.new } });
      message.success('密码已修改');
      form.resetFields();
      onClose();
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal title="修改密码" open={open} onCancel={onClose} onOk={() => form.submit()} confirmLoading={loading} destroyOnClose width={420}>
      <Form form={form} layout="vertical" onFinish={onFinish} style={{ marginTop: 8 }}>
        <Form.Item name="old" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}>
          <Input.Password placeholder="当前密码" />
        </Form.Item>
        <Form.Item name="new" label="新密码" rules={[{ required: true, min: 8, message: '新密码至少 8 位' }]}>
          <Input.Password placeholder="至少 8 位" />
        </Form.Item>
        <Form.Item
          name="confirm"
          label="确认新密码"
          dependencies={['new']}
          rules={[
            { required: true, message: '请再次输入新密码' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('new') === value) return Promise.resolve();
                return Promise.reject(new Error('两次输入不一致'));
              },
            }),
          ]}
        >
          <Input.Password placeholder="再次输入新密码" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
