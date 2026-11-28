# 历史密钥扫描复核

2026-11-28，Gitleaks v8.30.1 扫描公开仓库116个提交，命中12条，全部来自初始导入提交4f20c9896961fb3eab82851be97c324c82636350。使用脱敏JSON报告及该提交源码逐项核对，未在这12条中发现运行凭据；不据此断言仓库从未泄露过其他密钥。

|位置|条数|判断依据|
|---|---:|---|
|docker-image-nightly.yml 105/106|2|Docker镜像公开标签，非认证信息。|
|common/str.go 250|1|注释中的脱敏格式示例，长度47字符；不是运行配置，也不是39字符完整GCP Key。|
|controller/token_test.go 395/423/450/475|4|seedToken向单元测试本地数据库写入的合成token，未供上游认证或部署。|
|middleware/auth.go 148|1|公开响应的版本兼容标识，非访问令牌。|
|vertex/service_account.go 68|1|PEM边界文本处理，只包含BEGIN/END标签，没有私钥内容。|
|jimeng/adaptor.go 226/410/412|3|请求体req_key的公开模型/任务标识，真实鉴权使用另行传入的配置。|

`.gitleaksignore` 仅精确豁免以上提交+文件+规则+行号组成的12个fingerprint。没有按文件、测试目录、规则或字符串内容全面放行，没有删除或改写Git历史。未来提交即使在同一文件/行出现新凭据，也不会被这些历史fingerprint豁免。保留现有默认规则与全历史扫描。

CI报告只输出RuleID/File/StartLine/Commit/Fingerprint，完整报告用redact=100生成于临时目录，不打印Secret/Match字段。不能为了“绿灯”将未知命中加入豁免；真实凭据需先撤销/轮换，再评估历史清理，不能只删除当前文件。

回归验证：`python scripts/test_secret_scan.py --gitleaks-command <gitleaks路径>` 检查12个精确历史fingerprint，并在项目临时目录生成从未注册/部署的合成值，确认新值触发generic-api-key失败。临时目录校验位于工作区内并在完成后清理，报告脱敏，不打印合成值；该回归也纳入CI。全历史本机复扫116个提交未再检出未豁免命中。
