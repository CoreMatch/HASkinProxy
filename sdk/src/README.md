// HASkinProxy SDK 页面源码目录。
//
// 本目录是编译时 SDK 包（compile-time SDK aggregation）的源，遵守
// HA-Contract/docs/dev/HRPAuth/sdk-package.md：
//   - manifest.json 声明前端集成面（路由 / 菜单 / Dashboard）；
//   - src/ 下的 TSX 模块被 HA-WebUI-SDKHandler 注入到
//     HRPAuth-Web/src/generated/services/haskinproxy/ 并编译进前端；
//   - 每个路由模块必须有 default export 的 React 组件（配合
//     React.lazy 懒加载）。
//
// 运行时约束：
//   - 组件只能使用 HRPAuth-Web 已有的依赖（React 19、MUI 7、
//     react-router-dom 7）——SDK 包声明新依赖需同步更新本目录的
//     manifest.dependencies，并确保 npm 可解析；
//   - HRPAuth 在页面启动时注入 window.__BACKEND_URL__（主服务回源
//     地址），CSL 配置页据此推导 CustomSkinAPI 根地址；CSS/静态资源
//     必须内联，SDK 包不允许携带外部 URL。
