# WebP服务器内存优化解决方案

## 问题分析

原WebP服务器在处理并发转换请求时存在以下问题：

1. **无限制并发**：所有转换请求同时启动，没有并发控制
2. **内存泄漏风险**：libvips在高并发下内存管理不当
3. **OOM风险**：5个并发请求可能导致内存使用超过500MB
4. **阻塞请求**：转换过程阻塞后续请求处理

## 解决方案

### 1. 内存管理器 (encoder/memory_manager.go)

创建了专门的内存管理器来控制并发转换：

- **并发限制**：限制最大同时转换数量（默认6个）
- **内存限制**：设置转换过程内存使用上限（默认150MB）
- **任务队列**：使用有界队列缓存待转换任务
- **工作池模式**：使用固定数量的工作协程处理转换任务

```go
type MemoryManager struct {
    maxConcurrency int
    currentJobs    int
    jobQueue       chan *ConversionJob
    semaphore      chan struct{}
    memoryLimitMB  int64
    currentMemory  int64
}
```

### 2. 转换流程优化 (encoder/encoder.go)

将原有的`ConvertFilter`函数重构：

- **异步提交**：转换任务提交给内存管理器
- **同步处理**：内存管理器内部使用工作池同步处理
- **资源控制**：通过信号量控制并发数量

### 3. 配置参数增强 (config/config.go)

新增配置参数：

- `MAX_CONCURRENT_CONVERSIONS`: 最大并发转换数（默认6）
- `MEMORY_LIMIT_MB`: 转换内存限制（默认150MB）

### 4. 预转换优化 (encoder/prefetch.go)

预转换过程使用内存管理器：

- **队列化处理**：预转换任务通过队列处理
- **内存控制**：避免预转换时内存爆炸
- **进度监控**：保持原有的进度条功能

### 5. 状态监控 (handler/router.go)

添加内存使用状态监控：

- **响应头信息**：`X-Memory-Jobs`, `X-Memory-Usage-MB`, `X-Queue-Size`
- **实时状态**：客户端可以监控服务器内存状态
- **调试支持**：便于运维人员监控和调试

## 配置说明

### config.json 示例

```json
{
  "HOST": "127.0.0.1",
  "PORT": "3333",
  "QUALITY": "80",
  "IMG_PATH": "./pics",
  "EXHAUST_PATH": "./exhaust",
  "ALLOWED_TYPES": ["jpg","png","jpeg","gif","bmp","svg","heic","nef"],
  "CONVERT_TYPES": ["webp"],
  "STRIP_METADATA": true,
  "ENABLE_EXTRA_PARAMS": false,
  "READ_BUFFER_SIZE": 4096,
  "CONCURRENCY": 262144,
  "DISABLE_KEEPALIVE": false,
  "CACHE_TTL": 259200,
  "MAX_CACHE_SIZE": 0,
  "MAX_CONCURRENT_CONVERSIONS": 6,
  "MEMORY_LIMIT_MB": 150
}
```

### 环境变量支持

```bash
# 最大并发转换数
export WEBP_MAX_CONCURRENT_CONVERSIONS=6

# 内存限制（MB）
export WEBP_MEMORY_LIMIT_MB=150
```

## 性能改进效果

### 内存使用控制

- **原来**：5个并发请求 → >500MB内存
- **现在**：最多6个并发，内存限制150MB
- **OOM风险**：从高风险 → 低风险

### 并发控制

- **原来**：无限制并发，系统过载
- **现在**：有界队列 + 工作池，平滑处理

### 请求处理

- **原来**：转换请求阻塞后续请求
- **现在**：任务队列化，非阻塞提交

## 监控和调试

### 响应头监控

每个响应都会包含内存状态信息：

```
X-Memory-Jobs: 2              # 当前正在处理的转换任务数
X-Memory-Usage-MB: 45         # 当前预估内存使用量(MB)
X-Queue-Size: 3               # 队列中等待的任务数
```

### 日志监控

系统会输出详细的内存管理日志：

```
INFO[0001] MemoryManager initialized: max_concurrency=6, memory_limit=150MB
DEBUG[0002] Job submitted to queue: /path/to/image.jpg
DEBUG[0003] Job completed. Current jobs: 2, Memory usage: 45MB
```

## 部署建议

### 1GB内存服务器配置

```json
{
  "MAX_CONCURRENT_CONVERSIONS": 4,
  "MEMORY_LIMIT_MB": 120
}
```

### 2GB内存服务器配置

```json
{
  "MAX_CONCURRENT_CONVERSIONS": 8,
  "MEMORY_LIMIT_MB": 200
}
```

## 使用方式

### 编译和运行

```bash
go build -o webp-server .
./webp-server -config config.json
```

### 预转换模式

```bash
# 后台预转换
./webp-server -prefetch -config config.json

# 前台预转换完成后退出
./webp-server -prefetch-foreground -config config.json
```

## 验证方法

1. **内存监控**：使用`htop`或`top`观察内存使用
2. **并发测试**：使用`ab`或`wrk`进行并发请求测试
3. **状态检查**：检查响应头中的内存状态信息

这个解决方案通过引入内存管理器和并发控制，有效解决了WebP服务器在高并发场景下的内存消耗问题，确保服务器在1GB内存的虚拟机上稳定运行。