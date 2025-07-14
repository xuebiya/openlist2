package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// MediaLogger 是一个专门记录媒体文件访问的日志中间件
// 它会完全替代原有的日志系统

// 支持的媒体文件扩展名
var mediaExtensions = map[string]bool{
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

// 要忽略的路径前缀
var ignoredPaths = []string{
	"/assets/",
	"/images/",
	"/favicon.ico",
	"/robots.txt",
	"/ping",
}

// 请求和响应的结构体，用于解析JSON
type fsRequest struct {
	Path string `json:"path"`
}

type fsObject struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type int    `json:"type"`
}

type fsListResponse struct {
	Code    int       `json:"code"`
	Content []fsObject `json:"content"`
}

type fsGetResponse struct {
	Code    int     `json:"code"`
	Data    fsObject `json:"data"`
}

// MediaLoggerMiddleware 返回一个只记录媒体文件访问的日志中间件
func MediaLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果是静态资源或其他忽略的路径，直接跳过
		path := c.Request.URL.Path
		for _, prefix := range ignoredPaths {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}

		// 检查是否是直接访问媒体文件的路径
		if isMediaFilePath(path) {
			// 记录直接访问媒体文件的日志
			start := time.Now()
			c.Next()
			latency := time.Since(start)
			
			clientIP := c.ClientIP()
			method := c.Request.Method
			statusCode := c.Writer.Status()
			
			log.Infof("[GIN] %v | %3d | %13v | %15s | %-7s %s | 直接访问媒体文件",
				time.Now().Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				method,
				path,
			)
			return
		}

		// 检查是否是API调用
		if strings.HasPrefix(path, "/api/") {
			// 如果是 /api/fs/list 或 /api/fs/get，需要特殊处理
			if path == "/api/fs/list" || strings.HasPrefix(path, "/api/fs/list?") {
				handleFSListRequest(c)
				return
			} else if path == "/api/fs/get" || strings.HasPrefix(path, "/api/fs/get?") {
				handleFSGetRequest(c)
				return
			}
			
			// 其他API调用不记录日志
			c.Next()
			return
		}

		// 默认情况下不记录日志
		c.Next()
	}
}

// 处理 /api/fs/list 请求
func handleFSListRequest(c *gin.Context) {
	// 保存请求体
	var requestBody []byte
	if c.Request.Body != nil {
		requestBody, _ = io.ReadAll(c.Request.Body)
		// 恢复请求体，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
	}

	// 创建响应体捕获器
	responseWriter := &responseBodyWriter{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
	}
	c.Writer = responseWriter

	// 记录开始时间
	start := time.Now()
	
	// 处理请求
	c.Next()
	
	// 计算延迟时间
	latency := time.Since(start)
	
	// 检查请求体中是否包含媒体文件路径
	var req fsRequest
	if len(requestBody) > 0 {
		_ = json.Unmarshal(requestBody, &req)
	}

	// 检查响应体中是否包含媒体文件
	responseData := responseWriter.body.Bytes()
	var resp fsListResponse
	if len(responseData) > 0 {
		_ = json.Unmarshal(responseData, &resp)
	}

	// 检查响应中是否包含媒体文件
	hasMediaFile := false
	mediaFiles := []string{}
	
	if resp.Code == 200 && len(resp.Content) > 0 {
		for _, item := range resp.Content {
			if isMediaFile(item.Name) {
				hasMediaFile = true
				mediaFiles = append(mediaFiles, item.Name)
			}
		}
	}

	// 如果包含媒体文件，记录日志
	if hasMediaFile {
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := responseWriter.Status()
		
		log.Infof("[GIN] %v | %3d | %13v | %15s | %-7s %s | 目录: %s | 媒体文件: %v",
			time.Now().Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			c.Request.URL.Path,
			req.Path,
			mediaFiles,
		)
	}
}

// 处理 /api/fs/get 请求
func handleFSGetRequest(c *gin.Context) {
	// 保存请求体
	var requestBody []byte
	if c.Request.Body != nil {
		requestBody, _ = io.ReadAll(c.Request.Body)
		// 恢复请求体，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
	}

	// 创建响应体捕获器
	responseWriter := &responseBodyWriter{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
	}
	c.Writer = responseWriter

	// 记录开始时间
	start := time.Now()
	
	// 处理请求
	c.Next()
	
	// 计算延迟时间
	latency := time.Since(start)
	
	// 检查请求体中是否包含媒体文件路径
	var req fsRequest
	if len(requestBody) > 0 {
		_ = json.Unmarshal(requestBody, &req)
	}

	// 检查响应体中是否包含媒体文件
	responseData := responseWriter.body.Bytes()
	var resp fsGetResponse
	if len(responseData) > 0 {
		_ = json.Unmarshal(responseData, &resp)
	}

	// 检查响应中是否包含媒体文件
	if resp.Code == 200 && isMediaFile(resp.Data.Name) {
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := responseWriter.Status()
		
		log.Infof("[GIN] %v | %3d | %13v | %15s | %-7s %s | 访问媒体文件: %s | 路径: %s",
			time.Now().Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			c.Request.URL.Path,
			resp.Data.Name,
			resp.Data.Path,
		)
	}
}

// 检查路径是否为媒体文件
func isMediaFilePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return mediaExtensions[ext]
}

// 检查文件名是否为媒体文件
func isMediaFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return mediaExtensions[ext]
}

// responseBodyWriter 是一个用于捕获响应体的包装器
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 实现 ResponseWriter 接口
func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString 实现 ResponseWriter 接口
func (w *responseBodyWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Status 获取状态码
func (w *responseBodyWriter) Status() int {
	return w.ResponseWriter.Status()
}

// 启用调试模式的日志记录器
func MediaLoggerWithDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录所有请求的开始信息
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		
		// 获取请求体
		var requestBody []byte
		if c.Request.Body != nil && c.Request.Method != "GET" {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// 恢复请求体，以便后续处理
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		
		// 记录请求信息
		if raw != "" {
			path = path + "?" + raw
		}
		
		log.Debugf("[请求] %s %s", c.Request.Method, path)
		if len(requestBody) > 0 {
			log.Debugf("[请求体] %s", string(requestBody))
		}
		
		// 创建响应体捕获器
		responseWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = responseWriter
		
		// 记录开始时间
		start := time.Now()
		
		// 处理请求
		c.Next()
		
		// 计算延迟时间
		latency := time.Since(start)
		
		// 检查是否为媒体文件访问
		isMedia := false
		
		// 检查路径
		if isMediaFilePath(path) {
			isMedia = true
		}
		
		// 检查请求体
		if !isMedia && len(requestBody) > 0 {
			reqStr := strings.ToLower(string(requestBody))
			for ext := range mediaExtensions {
				if strings.Contains(reqStr, ext) {
					isMedia = true
					break
				}
			}
		}
		
		// 检查响应体
		responseData := responseWriter.body.Bytes()
		if !isMedia && len(responseData) > 0 {
			respStr := strings.ToLower(string(responseData))
			for ext := range mediaExtensions {
				if strings.Contains(respStr, ext) {
					isMedia = true
					break
				}
			}
		}
		
		// 记录响应信息
		statusCode := responseWriter.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		
		if isMedia {
			log.Infof("[GIN] %v | %3d | %13v | %15s | %-7s %s | 媒体文件访问",
				time.Now().Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				method,
				path,
			)
			
			// 输出更详细的调试信息
			log.Debugf("[响应] 状态码: %d, 延迟: %v", statusCode, latency)
			if len(responseData) > 0 && len(responseData) < 1000 {
				log.Debugf("[响应体] %s", string(responseData))
			} else if len(responseData) >= 1000 {
				log.Debugf("[响应体] (长度: %d) %s...", len(responseData), string(responseData[:1000]))
			}
		} else {
			// 对于非媒体文件访问，只记录调试信息
			log.Debugf("[响应] %s %s | 状态码: %d, 延迟: %v", method, path, statusCode, latency)
		}
	}
} 