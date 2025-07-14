# OpenList 日志过滤器

这个日志过滤器是为了优化 OpenList 的日志输出而设计的，它可以过滤掉不必要的日志信息，只保留重要的媒体文件访问记录。

## 功能说明

1. **只记录媒体文件相关的日志**：
   - 支持常见的图片格式（jpg, png, gif等）
   - 支持常见的视频格式（mp4, avi, mkv等）
   - 当通过 `/api/fs/list` 或 `/api/fs/get` API 访问这些文件时，记录相关日志

2. **过滤掉不必要的日志**：
   - 静态资源访问（/assets/, /images/, favicon.ico 等）
   - 不包含媒体文件的 API 调用
   - 其他不重要的请求

## 配置说明

该过滤器已经在应用的主要入口点配置完成，无需额外设置。它会自动检测请求中的媒体文件路径，并相应地过滤日志输出。

## 日志格式

保留的日志格式与原始 Gin 日志格式相同：

```
[GIN] 2025/07/12 - 15:10:36 | 200 | 2.656541586s | 10.26.0.4 | POST "/api/fs/list"
```

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

## 自定义扩展

如果需要支持更多的媒体文件格式，可以在 `server/middlewares/log_filter.go` 文件中的 `supportedExtensions` 变量中添加。 