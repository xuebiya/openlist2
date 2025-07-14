# OpenList 媒体日志系统

这是一个专门为 OpenList 项目设计的媒体文件日志系统，它只记录与视频和图片文件相关的访问日志，过滤掉所有其他类型的请求日志。

## 主要功能

1. **专注于媒体文件**：
   - 只记录视频和图片文件的访问日志
   - 支持多种常见媒体文件格式（如 mp4、jpg、png 等）
   - 完全忽略其他类型的请求日志

2. **深度检测**：
   - 直接检查请求路径中的媒体文件扩展名
   - 解析 `/api/fs/list` 和 `/api/fs/get` 请求的响应内容
   - 检查响应中是否包含媒体文件信息

3. **详细日志格式**：
   - 记录访问时间、状态码、延迟时间、客户端 IP、请求方法和路径
   - 对于目录列表，记录包含的媒体文件名
   - 对于单个文件访问，记录文件名和路径

4. **调试模式**：
   - 提供详细的请求和响应信息
   - 记录请求体和响应体内容
   - 帮助排查问题

## 支持的媒体文件格式

### 图片格式
- jpg/jpeg
- png
- gif
- bmp
- webp
- svg
- tiff
- ico
- heic

### 视频格式
- mp4
- avi
- mkv
- mov
- wmv
- flv
- webm
- m4v
- mpg/mpeg
- 3gp
- rm/rmvb
- ts
- m3u8

## 工作原理

1. **直接文件访问**：
   - 检查请求路径是否以支持的媒体文件扩展名结尾
   - 如果是，记录访问日志

2. **API 调用**：
   - 对于 `/api/fs/list` 请求：
     - 捕获请求体和响应体
     - 解析响应中的文件列表
     - 检查是否包含媒体文件
     - 如果包含，记录日志并列出媒体文件名
   
   - 对于 `/api/fs/get` 请求：
     - 捕获请求体和响应体
     - 解析响应中的文件信息
     - 检查是否为媒体文件
     - 如果是，记录日志并显示文件名和路径

3. **其他请求**：
   - 完全忽略，不记录日志

## 使用方式

系统已经在服务器入口处配置好，无需额外设置。它会自动根据是否为调试模式选择合适的日志中间件：

```go
if flags.Debug || flags.Dev {
    r.Use(middlewares.MediaLoggerWithDebug(), gin.RecoveryWithWriter(log.StandardLogger().Out))
} else {
    r.Use(middlewares.MediaLoggerMiddleware(), gin.RecoveryWithWriter(log.StandardLogger().Out))
}
```

## 日志输出示例

### 直接访问媒体文件
```
[GIN] 2025/07/12 - 15:10:36 | 200 | 2.656541586s | 10.26.0.4 | GET     /d/path/to/video.mp4 | 直接访问媒体文件
```

### 目录列表中包含媒体文件
```
[GIN] 2025/07/12 - 15:10:43 | 200 | 2.470475732s | 10.26.0.4 | POST     /api/fs/list | 目录: /movies | 媒体文件: [video1.mp4 video2.mkv]
```

### 获取单个媒体文件信息
```
[GIN] 2025/07/12 - 15:11:01 | 200 | 2.742716744s | 10.26.0.4 | POST     /api/fs/get | 访问媒体文件: movie.mp4 | 路径: /movies/movie.mp4
```

## 自定义扩展

如果需要支持更多的媒体文件格式，可以在 `server/middlewares/media_logger.go` 文件中的 `mediaExtensions` 变量中添加。

## 调试模式

在调试模式下，系统会输出更多详细信息，包括请求体和响应体内容，帮助排查问题。要启用调试模式，只需设置 `flags.Debug` 或 `flags.Dev` 为 `true`。 