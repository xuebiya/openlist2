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
	log "github.com/sirupsen/logrus"
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

// 要过滤的API路径
var apiPathsToFilter = []string{
	"/api/fs/list",
	"/api/fs/get",
	"/api/me",
	"/api/public",
	"/api/auth",
	"/api/admin",
	"/api/fs/other",
	"/api/fs/dirs",
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
	apiRegexes      []*regexp.Regexp
	pathRegex       *regexp.Regexp
	regexInitOnce   sync.Once
	debugMode       = false // 设置为true可以输出调试信息
)

// 初始化正则表达式
func initRegex() {
	regexInitOnce.Do(func() {
		// 为每个API路径创建正则表达式
		apiRegexes = make([]*regexp.Regexp, len(apiPathsToFilter))
		for i, path := range apiPathsToFilter {
			// 匹配任何HTTP方法访问该API路径
			apiRegexes[i] = regexp.MustCompile(`\s+"` + path)
		}
		
		// 构建媒体文件扩展名模式
		extensionsPattern := strings.Join(getMediaExtensions(), "|")
		
		// 更宽松的媒体文件路径匹配模式
		pathRegex = regexp.MustCompile(`[^a-zA-Z0-9](` + extensionsPattern + `)[^a-zA-Z0-9]`)
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
		// 检查是否是需要过滤的API调用
		isApiCall := false
		for _, regex := range apiRegexes {
			if regex.MatchString(logLine) {
				isApiCall = true
				break
			}
		}
		
		// 如果是API调用
		if isApiCall {
			// 检查是否包含媒体文件扩展名
			hasMediaExt := false
			for ext := range supportedExtensions {
				if strings.Contains(strings.ToLower(logLine), ext) {
					hasMediaExt = true
					break
				}
			}
			
			// 使用正则表达式再次检查
			if !hasMediaExt {
				hasMediaExt = pathRegex.MatchString(strings.ToLower(logLine))
			}
			
			// 如果包含媒体文件扩展名，记录日志
			if hasMediaExt {
				if debugMode {
					log.Debugf("记录媒体文件API访问日志: %s", logLine)
				}
				return w.Writer.Write(p)
			}
			
			// 没有找到媒体文件路径，忽略此日志
			if debugMode {
				log.Debugf("过滤API访问日志: %s", logLine)
			}
			return len(p), nil
		}
		
		// 如果是静态资源路径，不记录
		if strings.Contains(logLine, "/assets/") || 
		   strings.Contains(logLine, "/images/") || 
		   strings.Contains(logLine, "/favicon.ico") {
			return len(p), nil
		}
		
		// 检查是否是直接访问媒体文件的路径
		for ext := range supportedExtensions {
			if strings.Contains(strings.ToLower(logLine), ext) {
				if debugMode {
					log.Debugf("记录直接媒体文件访问日志: %s", logLine)
				}
				return w.Writer.Write(p)
			}
		}
		
		// 其他 GIN 日志行记录
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
			
			path := param.Path
			
			// 检查是否为API调用
			isApiCall := false
			for _, apiPath := range apiPathsToFilter {
				if strings.HasPrefix(path, apiPath) {
					isApiCall = true
					break
				}
			}
			
			// 如果是API调用，检查是否包含媒体文件路径
			if isApiCall {
				// 从请求体、URL或响应中检查媒体文件扩展名
				hasMediaExt := false
				
				// 检查URL
				urlStr := param.Request.URL.String()
				for ext := range supportedExtensions {
					if strings.Contains(strings.ToLower(urlStr), ext) {
						hasMediaExt = true
						break
					}
				}
				
				// 如果没有找到媒体文件扩展名，不记录
				if !hasMediaExt {
					return ""
				}
				
				if debugMode {
					log.Debugf("记录媒体文件API访问: %s", path)
				}
			}
			
			// 忽略静态资源路径
			if strings.Contains(path, "/assets/") || 
			   strings.Contains(path, "/images/") || 
			   strings.Contains(path, "/favicon.ico") {
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

// EnableDebugMode 启用调试模式，输出更多日志信息
func EnableDebugMode() {
	debugMode = true
	log.Info("日志过滤器调试模式已启用")
}

// DisableDebugMode 禁用调试模式
func DisableDebugMode() {
	debugMode = false
} 