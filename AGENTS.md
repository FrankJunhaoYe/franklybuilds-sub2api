# FranklyBuilds Sub2API 项目工作约定

## 项目定位与维护目标

本仓库是 FranklyBuilds 为自己的 Sub2API API 中转站维护的二次开发项目。继承上游的 API 网关、账号池、鉴权、路由、计费、配额、支付与管理功能，并维护自己的界面、品牌和后续业务定制。

- `origin`：`https://github.com/FrankJunhaoYe/franklybuilds-sub2api.git`，二开仓库。
- `upstream`：`https://github.com/Wei-Shaw/sub2api.git`，原项目；默认跟进 `upstream/main`。
- 长期目标是跟上上游绝大多数更新，同时保留 FranklyBuilds 的修改。不要把二开维护成冻结的旧版本，也不要用上游整树覆盖来完成更新。
- 默认完整合并选定上游版本的历史，包括功能、修复、安全和依赖更新。只有明确不适用或存在兼容性问题的改动才单独适配、暂缓或回退；记录对应提交、理由、影响和后续处理条件。
- 优先通过配置、独立定制模块和少量接入点实现差异，避免大面积改名、格式化、搬移上游代码。

本文规定本仓库的二开流程。`README*.md` 和 `DEV_GUIDE.md` 可用于了解上游，但不能据此推断本项目的分支、服务器或凭据。当前 `DEV_GUIDE.md` 含其他 fork 的名称、机器配置和过时工具版本；运行命令以当前 `go.mod`、`package.json`、Makefile 和 CI 配置为准。

## 三个分支的职责

这里指 `origin` 下的三个分支；`upstream` 的其他功能分支不属于本项目的三分支约定。

| 分支 | 长期职责 | 工作规则 |
| --- | --- | --- |
| `upstream-baseline` | 记录已接入二开主线的纯上游版本，供比较和下次同步使用 | 只指向真实上游提交；不加入主题、业务定制或本项目的工作文档。通常仅快进，不混入二开合并提交。 |
| `main` | FranklyBuilds 的集成主线，承载已验收的上游更新和定制功能 | 从功能分支和同步分支合入。它是后续发布的代码来源，分支存在本身不代表已经部署。 |
| `feature/frankly-warm-theme` | FranklyBuilds 暖色主题的开发分支 | 保留主题与字体定制，验证后合入 `origin/main`；继续开发时吸收主线更新。其他新功能另建主题明确的功能分支。 |

### 已核实的状态（2026-10-30）

- `origin/upstream-baseline` 位于 `a3298b46e`。
- `origin/main` 位于 `4ecf5397b`，其文件树与上述基线完全一致；历史额外包含初始化及合入基线的提交，**尚未包含暖色主题**。
- `origin/feature/frankly-warm-theme` 位于 `e92e90d28`，比 `origin/main` 多 6 个主题提交，涉及 32 个文件，全部位于 `frontend/`；这批提交没有修改后端、前端依赖清单或锁文件。
- 核查时 `upstream/main` 位于 `d851a3847`，比基线多 10 个提交。这是当时的同步差距，不代表这些更新已经合入本项目。
- 本地当前分支为 `feature/frankly-warm-theme`。本地 `main` 位于基线提交，**跟踪 `upstream/main`，而非 `origin/main`**；`upstream-baseline` 当时仅有远程跟踪引用。

以上是核查快照，开始任务时重新检查。不能用本地同名 `main` 代替 `origin/main`，不能在未核对 tracking 的情况下执行无参数 `git pull` 或 `git push`。本次新增工作文档也不等于主题分支已经合入主线。

## 开始任务前

在实际 Git 仓库根目录运行命令；不要把外层同名容器目录当作仓库。

```sh
git rev-parse --show-toplevel
git status --short
git branch -a -vv
git remote -v
```

- 保留用户的未提交修改；需要隔离工作时使用独立分支或 worktree，不擅自清理、覆盖或重置。
- 同步前先 `git fetch origin` 和 `git fetch upstream main`，核对远程地址及分支指向。
- 一般新功能从最新 `origin/main` 建分支；继续暖色主题则在现有主题分支上工作。不要仅因为上游基线更“干净”就从基线重新开发并丢掉已有定制。
- 阅读涉及目录的局部约定，以及与任务有关的现有实现、测试和文档。区分上游已有能力与 FranklyBuilds 新增能力。

## 跟进上游的流程

默认采用保留历史的 merge 流程。不要把长期同步变成逐条挑选少量补丁，也不要 rebase 已共享主线、强推公共分支或将主线 reset 到上游。

1. **确定同步范围。** fetch 后记录当前基线、`origin/main`、目标上游提交和仍未合入的功能分支。先确认旧基线是目标上游的祖先；若不是，调查分叉原因，不强制移动基线。
2. **检查改动。** 阅读旧基线到目标提交的日志和差异，重点查看依赖、配置默认值、API、迁移、鉴权、计费及主题接入点。同步通常接受选定版本的整体更新。
3. **隔离合并。** 从 `origin/main` 建立 `sync/upstream-YYYYMMDD`（重名时加后缀），将已核对的目标上游 SHA 合入该分支。目标使用固定 SHA，避免同步过程中远程变化导致范围漂移。
4. **保留行为并适配。** 同时保留上游修复和定制意图。不要对整棵树或所有冲突机械选择 `ours` / `theirs`。上游替换了组件或 API 时，将定制迁到新实现，并检查旧接入点是否失效。
5. **验证后集成。** 按下文执行相关检查，记录上游 SHA、冲突处理、保留的定制、例外和验证结果，再将同步分支合入二开 `main`。上游同步应保留可追溯的祖先关系，避免 squash 掉整次上游历史。
6. **推进基线。** 二开主线成功接入目标后，将 `upstream-baseline` 快进到同一个纯上游 SHA。不能把它指向包含二开改动的 merge commit，也不能在主线尚未接入时将其标为已同步。
7. **更新在途定制。** 将更新后的二开主线合入仍在开发的功能分支，再验证其定制。当前暖色主题尚未进入 `origin/main`，只同步主线不足以证明主题已经兼容新上游。

用于检查范围的命令（在 fetch 后执行）：

```sh
git rev-parse origin/upstream-baseline origin/main upstream/main
git merge-base --is-ancestor origin/upstream-baseline upstream/main
git log --oneline origin/upstream-baseline..upstream/main
git diff --stat origin/upstream-baseline upstream/main
git diff --stat origin/upstream-baseline origin/main
git diff --stat origin/main...origin/feature/frankly-warm-theme
```

`merge-base --is-ancestor` 返回 0 才表示祖先关系成立。创建同步分支时使用 `git switch --no-track -c sync/upstream-YYYYMMDD origin/main`；在干净工作区中执行 `git merge --no-ff --no-commit <已核对的上游SHA>`，检查、验证后再完成提交。尖括号内容是占位符，不能直接执行。

常规 Git 发布目标是 `origin`；只有任务明确涉及向原项目贡献时才操作上游 PR。是否提交、推送、合并或部署由当前任务范围决定，编写文档或检查分支不包含这些操作。

### 开发任务收尾（2026-11-03 起）

- 功能或修复开发完成并验证后，默认提交、推送至 `origin` 并创建 PR，不再只留下未提交改动。
- PR 创建后可以清理已推送的本地临时分支，保留可恢复的工作树；远程功能分支在 PR 成功合并后自动删除。创建 PR 本身不代表授权合并或部署，仍按当前任务授权范围执行。
- 检查并清理开发测试留下的多余容器、测试镜像、测试数据卷与构建缓存。先确认归属，保留仍用于验收的环境和数据；完成后说明清理范围及仍保留的资源。

## 必须保留的定制与接入点

当前已实现的定制是暖色界面和字体体系。它保留原有业务流程，没有独立实现另一套网关或计费后端。主题详细说明见 [frontend/src/franklybuilds/README.md](frontend/src/franklybuilds/README.md)。

| 路径 | 作用与合并时检查内容 |
| --- | --- |
| `frontend/src/franklybuilds/theme.js` | 色板和视觉效果参数；将上游常用 Tailwind 色系映射为暖色，修改会影响全站。 |
| `frontend/src/franklybuilds/theme.css` | 字体声明、共享表面、导航、认证及页面适配；弹窗/菜单存在 teleport 到 body 的情况，不能全部限制在布局容器内部。 |
| `frontend/src/franklybuilds/fonts/` | 西文字体、中文 Noto Sans SC 分片及相应许可文件；保留系统字体回退和按需加载。 |
| `frontend/tailwind.config.js` | 导入定制色板并配置字体、阴影、圆角；保留上游工具类命名，减少逐组件改写。 |
| `frontend/src/main.ts` | 在基础 `style.css` 之后导入 `franklybuilds/theme.css`；合并时检查入口与顺序。 |
| `frontend/src/style.css` | 共享按钮、卡片、输入框等样式；既要接入上游组件变化，也要保留主题语义。 |
| `frontend/src/components/`、`frontend/src/views/` 内的主题差异 | 布局和首页，以及认证、公告、模型广场、用户资料、仪表盘、运维和法律文档页面的样式/hooks；不能只保留独立主题目录而漏掉这些接入点。 |

维护主题时：

- 优先修改色板、共享样式和已有语义类，再使用局部 hook。不要为每个上游页面复制组件或堆叠一套工具类覆盖。
- 保留奶油色背景、克制的珊瑚色主操作、暖色暗黑模式、轻阴影及分层圆角。不要重新引入装饰性渐变、光晕、毛玻璃或模糊背景球。
- 保留主按钮 600 色阶、浅色次要文字和暗色文字的对比度处理；成功、警告、错误和禁用状态仍需可辨认。
- 支付渠道的品牌色保持原有辨识度；站点名称、Logo 等继续使用公开设置和 branding 逻辑，避免替换到上游 API 名称、协议标识、模块路径或第三方资源。
- 不要预加载全部中文字体分片。保留主题 README 对字体来源、许可状态和加载体积的说明，不把未确认事项写成已解决。
- 后续新增业务定制时，将其入口、数据/API 影响、与上游的交界点和验证方式补充到定制说明中，保证下次同步可以查清哪些行为必须保留。

## 代码结构与开发约定

- `backend/`：Go + Gin + Ent + Wire；`internal/handler` 处理 HTTP 接口，`internal/service` 放业务规则，`internal/repository` 访问数据，`ent/schema` 定义实体，`migrations` 放 SQL 迁移。
- `frontend/`：Vue 3 + TypeScript + Vite + Pinia + Tailwind CSS；复用 `src/api`、现有状态管理、公共组件和 i18n。
- `deploy/`、根目录 Dockerfile 和 `.github/workflows/`：部署与 CI。修改配置、依赖或运行时版本时，一并核对相关镜像、示例和流水线。
- 数据服务使用 PostgreSQL 和 Redis。连接信息来自任务环境和实际配置，不能把上游文档中的本机配置当作本项目凭据。
- 使用 pnpm；更改前端依赖时同步维护 `frontend/pnpm-lock.yaml`。不要引入另一套 npm/yarn 锁文件。
- Go 版本读取 `backend/go.mod`，Node/pnpm/lint 版本读取当前 CI。核查时为 Go 1.27.3、Node 20、pnpm 9、golangci-lint 2.14.0；这些版本会随上游更新。
- Ent/Wire 生成代码应通过 `backend/Makefile` 的 `generate` 流程更新，修改源 schema/provider 后检查生成差异，避免手改生成结果。
- 已应用的 SQL 迁移受 checksum 校验保护，不改写内容、编号或换行符；新增迁移需检查与上游编号碰撞，已上线的定制迁移发生冲突时设计兼容方案，不能直接重编号掩盖问题。
- 遵守 `.gitattributes`，SQL、Go、前端源码和脚本等使用规定的 LF；避免无关的全文件换行变更。
- 金额、余额、费率、折扣、配额及支付确认由后端校验、计算和授权。不能仅靠前端隐藏、输入限制或客户端计算来保证计费与权限正确。
- 保持网关协议、流式响应、错误处理和已有 API/数据库兼容性。改动这些行为时补充有意义的回归验证；日志不包含真实 Token、API Key、支付密钥或完整敏感请求。

## 验证命令与完成标准

按变更影响选择检查；纯文档修改只需核对事实、路径、Git 差异和格式，不为此运行整套应用测试。不要为低影响改动编写只复述实现的测试。

前端命令在仓库根目录运行：

```sh
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run test:run
pnpm --dir frontend run build
```

- 小范围逻辑修改可先用 `pnpm --dir frontend exec vitest run <测试文件>` 验证相关行为；整体主题或上游合并应运行前端测试和生产构建。
- `lint:check` 只检查；`lint` 带 `--fix`，会修改文件，不把它当作只读检查。
- `make test-frontend` 对应 CI 的 lint、类型检查和关键 Vitest 子集，不等于全部前端测试；完整测试使用 `test:run`，`test` 默认进入 watch 模式。
- `build` 包含 i18n 检查与类型构建，产物写入 `backend/internal/web/dist/`。不要误以为产物在 `frontend/dist/`，也不要提交生成的构建产物。
- 本地预览使用 `pnpm --dir frontend run dev`。Vite 默认端口 3000，代理目标为 `http://localhost:8080`，可由 `VITE_DEV_PORT` / `VITE_DEV_PROXY_TARGET` 覆盖；后端不可用时不能宣称完整业务已验证。

后端命令在 `backend/` 目录运行：

```sh
go test -tags=unit ./...
golangci-lint run --timeout=30m ./...
go test -tags=integration ./...
go build ./cmd/server
```

- 集成测试涉及 PostgreSQL/Redis 和 Testcontainers 等环境条件；缺少依赖或发生 skip 时说明实际覆盖范围，不把跳过视为验证成功。
- `make test` / `make test-backend` 不能代替显式的 unit、integration 两组 CI 检查；以 `.github/workflows/backend-ci.yml` 为准。
- Makefile 的 shell 配方依赖相应工具；Windows 下可用上述原生命令，涉及 POSIX 脚本时使用适合的环境，不将 shell 语法混用。
- 要构建包含前端的后端程序，先完成前端 build，再在 `backend/` 使用 `go build -tags embed ./cmd/server`。普通后端 build 不证明静态界面已打包。

主题修改或影响前端的上游更新，还应在浏览器检查首页、登录页、用户和管理员仪表盘、运维页、资料绑定、数据表格和 teleport 弹窗，覆盖亮/暗模式、移动宽度和字体实际加载。自动化测试通过不等于视觉检查通过。

交付时说明改动、保留的定制、实际执行的验证和未覆盖范围。同步任务还需说明接入的上游 SHA 及任何例外；没有执行部署就不宣称线上已更新。

## 文件署名与元数据

所有新建或编辑文件不添加作者栏、署名、工具署名或身份归属字段。生成或保存文档时，将支持的 author、creator、last-modified-by 等可编辑归属元数据留空，并检查产物；不要接受库自动填入的身份信息。项目名称可用于说明项目用途，不作为署名。
