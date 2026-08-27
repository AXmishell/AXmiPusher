/**
 * 站内信列表页。
 *
 * - FlatList 展示站内信(未读加粗 + 蓝点, 已读置灰);
 * - 下拉刷新 / 上拉加载更多(分页, id 倒序);
 * - 点击消息: 展开内容并标记已读(乐观更新, 失败回滚);
 * - 顶部未读数 + "全部已读"按钮; 新消息到达时顶部插入并横幅提示;
 * - 未读数同步到底部页签角标(tabBarBadge);
 * - 请求遇 401 时通知全局自动登出。
 */
import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs';
import { AuthExpiredError } from '../api/client';
import { listInbox, markAllRead, markRead, unreadCount } from '../api/inbox';
import type { InboxMessage } from '../api/types';
import { useAuth } from '../context/AuthContext';
import type { MainTabParamList } from '../navigation/types';
import { subscribe } from '../services/poller';
import { colors, spacing } from '../theme';

/** 分页大小。 */
const PAGE_SIZE = 20;

/** 时间格式化: ISO 字符串 → "YYYY-MM-DD HH:mm"。 */
function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso;
  }
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

type Props = BottomTabScreenProps<MainTabParamList, 'Inbox'>;

export default function InboxScreen({ navigation }: Props) {
  const { session, notifyAuthExpired } = useAuth();
  const serverUrl = session?.serverUrl ?? '';
  const token = session?.token ?? '';

  const [messages, setMessages] = useState<InboxMessage[]>([]);
  const [total, setTotal] = useState(0);
  const [unread, setUnread] = useState(0);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [newMsgBanner, setNewMsgBanner] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  // 消息 ref: 供轮询回调等异步逻辑读取最新值
  const messagesRef = useRef<InboxMessage[]>([]);
  const pageRef = useRef(1);
  const bannerTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  /** 提交消息列表(同步 ref 与 state)。 */
  const commitMessages = useCallback((next: InboxMessage[]) => {
    messagesRef.current = next;
    setMessages(next);
  }, []);

  // 未读数同步到 tab 角标
  useEffect(() => {
    navigation.setOptions({
      tabBarBadge: unread > 0 ? unread : undefined,
    });
  }, [navigation, unread]);

  /** 统一错误处理: 401 → 全局登出; 其它错误由调用方提示。 */
  const handleError = useCallback(
    (err: unknown) => {
      if (err instanceof AuthExpiredError) {
        notifyAuthExpired();
      }
    },
    [notifyAuthExpired],
  );

  /** 顶部横幅提示(3 秒后自动消失)。 */
  const showBanner = useCallback((text: string) => {
    setNewMsgBanner(text);
    if (bannerTimerRef.current) {
      clearTimeout(bannerTimerRef.current);
    }
    bannerTimerRef.current = setTimeout(() => setNewMsgBanner(null), 3000);
  }, []);

  /** 加载第一页(初始化 / 下拉刷新)。 */
  const loadFirstPage = useCallback(
    async (asRefresh = false) => {
      if (!serverUrl || !token) {
        return;
      }
      if (asRefresh) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      try {
        const [listRes, unreadRes] = await Promise.all([
          listInbox(serverUrl, token, { current: 1, pageSize: PAGE_SIZE }),
          unreadCount(serverUrl, token),
        ]);
        commitMessages(listRes.data);
        setTotal(listRes.total);
        setUnread(unreadRes);
        pageRef.current = 1;
        setHasMore(listRes.data.length < listRes.total);
        setLoadError(null);
      } catch (err) {
        handleError(err);
        setLoadError('加载失败, 请下拉重试');
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [serverUrl, token, commitMessages, handleError],
  );

  // 首次加载 / 登录后重新加载
  useEffect(() => {
    void loadFirstPage();
  }, [loadFirstPage]);

  /** 上拉加载更多。 */
  const loadMore = useCallback(async () => {
    if (!serverUrl || !token || loadingMore || !hasMore) {
      return;
    }
    setLoadingMore(true);
    try {
      const next = pageRef.current + 1;
      const listRes = await listInbox(serverUrl, token, {
        current: next,
        pageSize: PAGE_SIZE,
      });
      // 按 id 去重合并
      const merged = [...messagesRef.current];
      const seen = new Set(merged.map(m => m.id));
      for (const m of listRes.data) {
        if (!seen.has(m.id)) {
          merged.push(m);
          seen.add(m.id);
        }
      }
      commitMessages(merged);
      setTotal(listRes.total);
      pageRef.current = next;
      setHasMore(merged.length < listRes.total);
    } catch (err) {
      handleError(err);
    } finally {
      setLoadingMore(false);
    }
  }, [serverUrl, token, loadingMore, hasMore, commitMessages, handleError]);

  /** 订阅前台轮询: 新消息插入列表顶部 + 横幅; 未读数同步。 */
  useEffect(() => {
    const unsubscribe = subscribe({
      onNewMessages: (news: InboxMessage[]) => {
        const existing = new Set(messagesRef.current.map(m => m.id));
        const fresh = news.filter(m => !existing.has(m.id));
        if (fresh.length === 0) {
          return;
        }
        // 按 id 倒序插入列表顶部(最大 id 在最上)
        const sorted = [...fresh].sort((a, b) => b.id - a.id);
        commitMessages([...sorted, ...messagesRef.current]);
        setTotal(prev => prev + fresh.length);
        showBanner(`收到 ${fresh.length} 条新站内信`);
      },
      onUnreadCountChange: (count: number) => {
        setUnread(count);
      },
    });
    return unsubscribe;
  }, [commitMessages, showBanner]);

  // 卸载时清理横幅定时器
  useEffect(() => {
    return () => {
      if (bannerTimerRef.current) {
        clearTimeout(bannerTimerRef.current);
      }
    };
  }, []);

  /** 点击消息: 展开/收起内容; 未读则标记已读(乐观更新)。 */
  const onPressItem = useCallback(
    async (item: InboxMessage) => {
      setExpandedId(prev => (prev === item.id ? null : item.id));
      if (!item.is_read) {
        // 乐观更新为已读
        commitMessages(
          messagesRef.current.map(m =>
            m.id === item.id ? { ...m, is_read: true } : m,
          ),
        );
        setUnread(prev => Math.max(0, prev - 1));
        try {
          await markRead(serverUrl, token, item.id);
        } catch (err) {
          handleError(err);
          // 失败回滚
          commitMessages(
            messagesRef.current.map(m =>
              m.id === item.id ? { ...m, is_read: false } : m,
            ),
          );
          setUnread(prev => prev + 1);
        }
      }
    },
    [serverUrl, token, commitMessages, handleError],
  );

  /** 全部标记已读。 */
  const onMarkAllRead = useCallback(async () => {
    if (unread === 0) {
      return;
    }
    try {
      await markAllRead(serverUrl, token);
      commitMessages(
        messagesRef.current.map(m => (m.is_read ? m : { ...m, is_read: true })),
      );
      setUnread(0);
    } catch (err) {
      handleError(err);
      Alert.alert('操作失败', err instanceof Error ? err.message : '未知错误');
    }
  }, [serverUrl, token, unread, commitMessages, handleError]);

  /** 渲染单条站内信。 */
  const renderItem = useCallback(
    ({ item }: { item: InboxMessage }) => {
      const expanded = expandedId === item.id;
      return (
        <Pressable style={styles.card} onPress={() => void onPressItem(item)}>
          <View style={styles.cardHeader}>
            {!item.is_read && <View style={styles.unreadDot} />}
            <Text
              style={[
                styles.cardTitle,
                !item.is_read ? styles.cardTitleUnread : styles.cardTitleRead,
              ]}
              numberOfLines={expanded ? undefined : 2}
            >
              {item.title || '(无标题)'}
            </Text>
            <Text style={styles.cardTime}>{formatTime(item.created_at)}</Text>
          </View>
          {expanded && !!item.content && (
            <Text style={styles.cardContent}>{item.content}</Text>
          )}
          {!expanded && (
            <Text style={styles.cardPreview} numberOfLines={1}>
              {item.content || ''}
            </Text>
          )}
        </Pressable>
      );
    },
    [expandedId, onPressItem],
  );

  return (
    <View style={styles.container}>
      {/* 顶部信息栏: 未读数 + 全部已读 */}
      <View style={styles.listHeader}>
        <Text style={styles.unreadText}>未读 {unread} 条 · 共 {total} 条</Text>
        <Pressable onPress={() => void onMarkAllRead()} disabled={unread === 0}>
          <Text
            style={[styles.markAllText, unread === 0 && styles.markAllDisabled]}
          >
            全部已读
          </Text>
        </Pressable>
      </View>

      {/* 新消息横幅提示 */}
      {newMsgBanner != null && (
        <View style={styles.banner}>
          <Text style={styles.bannerText}>{newMsgBanner}</Text>
        </View>
      )}

      {loading ? (
        <View style={styles.center}>
          <ActivityIndicator size="large" color={colors.primary} />
        </View>
      ) : (
        <FlatList
          data={messages}
          keyExtractor={item => String(item.id)}
          renderItem={renderItem}
          contentContainerStyle={
            messages.length === 0 ? styles.emptyContainer : styles.listContent
          }
          refreshControl={
            <RefreshControl
              refreshing={refreshing}
              onRefresh={() => void loadFirstPage(true)}
              colors={[colors.primary]}
            />
          }
          onEndReached={() => void loadMore()}
          onEndReachedThreshold={0.3}
          ListEmptyComponent={
            <View style={styles.center}>
              <Text style={styles.emptyText}>
                {loadError ? loadError : '暂无站内信'}
              </Text>
            </View>
          }
          ListFooterComponent={
            loadingMore ? (
              <ActivityIndicator
                style={styles.footerLoading}
                color={colors.primary}
              />
            ) : undefined
          }
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  center: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: spacing.xl,
  },
  listHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    backgroundColor: colors.card,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border,
  },
  unreadText: {
    fontSize: 13,
    color: colors.text,
  },
  markAllText: {
    fontSize: 13,
    color: colors.primary,
  },
  markAllDisabled: {
    color: colors.muted,
    opacity: 0.5,
  },
  banner: {
    backgroundColor: '#eff6ff',
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.lg,
  },
  bannerText: {
    color: colors.primary,
    fontSize: 13,
  },
  listContent: {
    padding: spacing.lg,
  },
  emptyContainer: {
    flexGrow: 1,
    justifyContent: 'center',
  },
  emptyText: {
    color: colors.muted,
    fontSize: 14,
  },
  footerLoading: {
    marginVertical: spacing.md,
  },
  card: {
    backgroundColor: colors.card,
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.border,
    padding: spacing.md,
    marginBottom: spacing.md,
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  unreadDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: colors.unreadDot,
    marginRight: spacing.xs,
  },
  cardTitle: {
    flex: 1,
    fontSize: 15,
    marginRight: spacing.xs,
  },
  cardTitleUnread: {
    color: colors.text,
    fontWeight: 'bold',
  },
  cardTitleRead: {
    color: colors.muted,
    fontWeight: 'normal',
  },
  cardTime: {
    fontSize: 12,
    color: colors.muted,
  },
  cardPreview: {
    marginTop: spacing.xs,
    fontSize: 13,
    color: colors.muted,
  },
  cardContent: {
    marginTop: spacing.sm,
    fontSize: 14,
    lineHeight: 20,
    color: colors.text,
  },
});
