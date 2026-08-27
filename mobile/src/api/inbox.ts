/**
 * 站内信 API。
 * 契约(来自后端 Go 源码):
 * - GET /api/v1/inbox?current&pageSize&read → data={data:[...],total,success}
 * - GET /api/v1/inbox/unread-count → data={unread:N}
 * - PUT /api/v1/inbox/:id/read → data={ok:true}
 * - PUT /api/v1/inbox/read-all → data={ok:true}
 */
import { request } from './client';
import type { InboxListData, InboxMessage, UnreadCountData } from './types';

/** 收件箱列表查询参数。 */
export interface ListInboxParams {
  current?: number;
  pageSize?: number;
  /** 过滤条件: 仅未读 / 仅已读; 不传 = 全部。 */
  read?: boolean;
}

/** 收件箱列表(分页, 按 id 倒序)。 */
export async function listInbox(
  serverUrl: string,
  token: string,
  params: ListInboxParams = {},
): Promise<{ data: InboxMessage[]; total: number }> {
  const { current = 1, pageSize = 20, read } = params;
  const query: string[] = [`current=${current}`, `pageSize=${pageSize}`];
  if (read !== undefined) {
    query.push(`read=${read}`);
  }
  const data = await request<InboxListData>(
    serverUrl,
    `/inbox?${query.join('&')}`,
    { auth: token },
  );
  return { data: data.data, total: data.total };
}

/** 未读站内信数量。 */
export async function unreadCount(
  serverUrl: string,
  token: string,
): Promise<number> {
  const data = await request<UnreadCountData>(
    serverUrl,
    '/inbox/unread-count',
    { auth: token },
  );
  return data.unread;
}

/** 标记单条站内信为已读。 */
export async function markRead(
  serverUrl: string,
  token: string,
  id: number,
): Promise<void> {
  await request<{ ok: boolean }>(serverUrl, `/inbox/${id}/read`, {
    method: 'PUT',
    auth: token,
  });
}

/** 全部标记已读。 */
export async function markAllRead(
  serverUrl: string,
  token: string,
): Promise<void> {
  await request<{ ok: boolean }>(serverUrl, '/inbox/read-all', {
    method: 'PUT',
    auth: token,
  });
}
