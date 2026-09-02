# 服务端项目检测

## 检测目标

识别当前工作区是否为 Java、Node.js 或 Python 服务端项目，并判断适合的接入分支。

## Java 信号

- `pom.xml`、`build.gradle`、`settings.gradle`
- `src/main/java`
- Spring Boot 依赖或 `application.yml` / `application.properties`

## Node.js 信号

- `package.json`
- Express、NestJS、Koa、Fastify 等依赖
- `src/`、`routes/`、`controllers/`、`services/`、`.env` 或 config 目录

## Python 信号

- `pyproject.toml`、`requirements.txt`、`setup.py`、`Pipfile`
- FastAPI、Flask、Django 依赖
- `app/`、`src/`、`routers/`、`views/`、`services/`、`settings.py`

## 决策

- 多语言同时存在时，优先选择当前业务服务所在目录；无法判断时，向用户确认要接入哪个服务。
- 识别到项目后，读取对应语言 playbook。
- 没有识别到服务端项目时，不生成“已接入”的结论。
