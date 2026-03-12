# zygarde

Zygarde 是一个现代化、模块化的环境搭建与部署工具。它秉承宝可梦 Zygarde 所代表的理念，致力于在开发环境中维护“秩序”与“完整性”。

本项目旨在构建一个声明式、对开发者友好的工具，用于一键部署本地数据库环境。
用户只需通过简单的配置文件定义期望的数据库集群拓扑，工具就会自动生成并执行标准化的容器编排配置，从而屏蔽底层技术复杂性。

容器编排配置（第一阶段：Docker Compose，第二阶段：K8s）

核心模块：

### Template Manager

Template CRUD：负责模板的上传（Create）、读取（Read）、更新（Update）、删除（Delete）和列表查询（List）。
Template Parsing and Validation：在上传模板时，解析模板内容，提取其中定义的变量（例如 `{{ .Port }}` 或 metadata 中的变量），并校验模板语法及变量定义是否合法。
Template Information Provision：向 “Blueprint Manager” 提供模板内容和变量规范，用于编排和变量赋值。

Template Manager：负责管理“零部件”（templates）的仓库管理员。

### Blueprint Manager

Blueprint CRUD：负责 blueprint 的创建、读取、更新、删除和列表查询。
Blueprint Orchestration：管理一个 blueprint 由哪些 templates 组成。
Variable Management：管理 blueprint 中每个被引用 template 所使用的具体变量值。
Blueprint Rendering（核心）：根据 blueprint 定义获取所需模板，注入变量值，并将所有模板片段组合成最终完整的 `docker-compose.yaml` 文件。

Blueprint Manager：负责管理“蓝图”（blueprints）的设计师，按照蓝图将零部件组装成产品。

### Environment Manager

Environment CRUD and State Management：负责环境实例的创建、读取、更新和删除，并持久化记录每个环境的状态（例如 Creating、Running、Stopped、Error）。
Environment-Blueprint Association：记录每个环境实例是由哪个 blueprint 创建的。
Metadata Management：管理环境元数据，例如唯一 ID、名称、创建时间、访问端点（如生成的 IP 和端口）。
Status Querying：对外提供查询当前环境状态和信息的 API。

Environment Manager：负责管理“生产线”和“产品实例”（environments）的库存管理员，跟踪每个产品当前的状态。

### Deployment Engine

Executing Deployment Commands：执行 `docker-compose up -d`、`down`、`stop`、`start` 等命令。
Project Isolation：确保为每个 environment 生成的 `docker-compose.yaml` 都作为独立的 Docker Compose 项目运行（通过带唯一项目名的 `-p` 参数），以避免命名冲突。
Status Capture and Feedback：捕获命令执行结果（成功、失败、输出），并将结果反馈给调用方。
Future Extensibility：支持未来扩展到 Kubernetes 等其他编排平台。

Deployment Engine：装配流水线上的“机械臂”，只负责执行“装配”“暂停”“启动”“销毁”等实际动作。

### Coordinator / Unified Facade

Process Orchestration：作为系统的大脑，将上述各组件的能力串联起来，完成用户指令。
API Exposure：为 CLI 和 Web API 层提供统一的内核接口。
Error Handling and Transactional Guarantees：协调多个组件之间的调用，在发生错误时执行清理和状态回滚（例如创建失败后，通过 Deployment Engine 清理资源，并将环境状态更新为 `Error`）。

Coordinator：负责接收指令（用户命令）的“总工程师”，指挥仓库管理员查找蓝图和零部件，调度机械臂执行工作，并实时更新库存状态。
