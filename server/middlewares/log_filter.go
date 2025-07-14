package middlewares

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 支持的视频和图片文件扩展名
var supportedExtensions = map[string]bool{
	// 图片格式
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
	".svg":  true,
	".tiff": true,
	".ico":  true,
	".heic": true,
	
	// 视频格式
	".mp4":  true,
	".avi":  true,
	".mkv":  true,
	".mov":  true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
	".m4v":  true,
	".mpg":  true,
	".mpeg": true,
	".3gp":  true,
	".rm":   true,
	".rmvb": true,
	".ts":   true,
	".m3u8": true,
}

// 检查路径是否为支持的媒体文件
func isMediaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

// 获取所有媒体文件扩展名（不带点号）
func getMediaExtensions() []string {
	extensions := make([]string, 0, len(supportedExtensions))
	for ext := range supportedExtensions {
		// 去掉前面的点号
		extensions = append(extensions, ext[1:])
	}
	return extensions
}

// 用于匹配 API 路径的正则表达式
var (
	listApiRegex    *regexp.Regexp
	getApiRegex     *regexp.Regexp
	pathRegex       *regexp.Regexp
	regexInitOnce   sync.Once
)

// 初始化正则表达式
func initRegex() {
	regexInitOnce.Do(func() {
		listApiRegex = regexp.MustCompile(`POST\s+"/api/fs/list"`)
		getApiRegex = regexp.MustCompile(`POST\s+"/api/fs/get"`)
		
		// 构建媒体文件扩展名模式
		extensionsPattern := strings.Join(getMediaExtensions(), "|")
		pathRegex = regexp.MustCompile(`"([^"]+\.(` + extensionsPattern + `))"`)
	})
}

// LoggerFilterWriter 是一个包装了原始输出流的写入器，用于过滤日志
type LoggerFilterWriter struct {
	Writer io.Writer
}

func (w *LoggerFilterWriter) Write(p []byte) (n int, err error) {
	// 确保正则表达式已初始化
	initRegex()
	
	// 解析日志行
	logLine := string(p)
	
	// 检查这是否是一个 GIN 日志行
	if strings.Contains(logLine, "[GIN]") {
		// 检查是否是 API 调用
		isListApi := listApiRegex.MatchString(logLine)
		isGetApi := getApiRegex.MatchString(logLine)
		
		// 如果是 API 调用
		if isListApi || isGetApi {
			// 查找日志中是否包含媒体文件路径
			matches := pathRegex.FindStringSubmatch(logLine)
			if len(matches) > 0 {
				// 找到媒体文件路径，记录日志
				return w.Writer.Write(p)
			}
			
			// 没有找到媒体文件路径，忽略此日志
			return len(p), nil
		}
		
		// 如果是静态资源路径（通常包含 /assets/ 或 /images/），不记录
		if strings.Contains(logLine, "/assets/") || 
		   strings.Contains(logLine, "/images/") || 
		   strings.Contains(logLine, "/favicon.ico") {
			return len(p), nil
		}
		
		// 其他 GIN 日志行（非 API 调用和非静态资源）记录
		return w.Writer.Write(p)
	}
	
	// 非 GIN 日志行直接写入
	return w.Writer.Write(p)
}

// FilteredLogger 返回一个过滤了非媒体文件访问日志的中间件
func FilteredLogger(writer io.Writer) gin.HandlerFunc {
	// 确保正则表达式已初始化
	initRegex()
	
	filteredWriter := &LoggerFilterWriter{Writer: writer}
	
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			var statusColor, methodColor, resetColor string
			if param.IsOutputColor() {
				statusColor = param.StatusCodeColor()
				methodColor = param.MethodColor()
				resetColor = param.ResetColor()
			}

			if param.Latency > time.Minute {
				param.Latency = param.Latency.Truncate(time.Second)
			}
			
			// 检查是否为媒体文件路径的访问
			isMediaPath := false
			path := param.Path
			
			// 检查路径是否为 API 调用
			isListOrGetApi := strings.HasPrefix(path, "/api/fs/list") || 
			                 strings.HasPrefix(path, "/api/fs/get")
			
			if isListOrGetApi {
				// 对于 API 调用，检查请求体或 URL 是否包含媒体文件路径
				isMediaPath = containsMediaPath(param.Request.URL.String())
			} else {
				// 对于直接访问文件的请求，检查路径是否为媒体文件
				isMediaPath = isMediaFile(path)
			}
			
			// 忽略静态资源路径
			if strings.Contains(path, "/assets/") || 
			   strings.Contains(path, "/images/") || 
			   strings.Contains(path, "/favicon.ico") {
				return ""
			}
			
			// 如果是 API 调用但不包含媒体文件路径，不记录
			if isListOrGetApi && !isMediaPath {
				return ""
			}
			
			// 构建日志行
			return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %s\n%s",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				statusColor, param.StatusCode, resetColor,
				param.Latency,
				param.ClientIP,
				methodColor, param.Method, resetColor,
				param.Path,
				param.ErrorMessage,
			)
		},
		Output:    filteredWriter,
		SkipPaths: []string{"/assets/", "/images/", "/favicon.ico"},
	})
}

// 检查请求URL或日志行中是否包含媒体文件路径
func containsMediaPath(s string) bool {
	// 确保正则表达式已初始化
	initRegex()
	
	// 尝试使用正则表达式查找媒体文件路径
	matches := pathRegex.FindStringSubmatch(s)
	return len(matches) > 0
} 