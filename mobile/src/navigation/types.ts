/**
 * 导航参数表定义(供各屏幕与导航器共享类型)。
 */

/** 根导航(认证栈)。 */
export type RootStackParamList = {
  /** 启动加载页。 */
  Loading: undefined;
  /** 登录页。 */
  Login: undefined;
  /** 已登录主界面(底部 Tabs)。 */
  Main: undefined;
};

/** 底部页签。 */
export type MainTabParamList = {
  /** 站内信列表。 */
  Inbox: undefined;
  /** 设置。 */
  Settings: undefined;
};
